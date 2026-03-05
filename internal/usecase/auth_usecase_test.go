package usecase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"
	jwtpkg "go-standard/internal/pkg/jwt"
	"go-standard/internal/pkg/rediskey"
	"go-standard/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	redisMockPkg "github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ─── Fixture ────────────────────────────────────────────────────────────────

type authUCFix struct {
	uc        AuthUsecase
	repo      *mocks.MockUserRepository
	dbMock    sqlmock.Sqlmock
	redisMock redisMockPkg.ClientMock
	jwtMgr    *jwtpkg.Manager
}

func newAuthUCFix(t *testing.T, es *elasticsearch.Client) *authUCFix {
	t.Helper()
	if es == nil {
		es = esOK(`{"result":"created"}`)
	}

	sqlDB, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	gDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	rdb, rMock := redisMockPkg.NewClientMock()
	repo := mocks.NewMockUserRepository(t)
	jwtMgr := newJWTMgr(t)
	uc := NewAuthUsecase(gDB, repo, rdb, es, jwtMgr, zap.NewNop())

	return &authUCFix{uc: uc, repo: repo, dbMock: dbMock, redisMock: rMock, jwtMgr: jwtMgr}
}

func hashedPwd(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func newAuthUser(t *testing.T, email, plain string) *model.User {
	t.Helper()
	phone := "123-456-7890"
	return &model.User{
		ID:        uuid.New(),
		Email:     email,
		Password:  hashedPwd(t, plain),
		Name:      "Test User",
		Role:      "user",
		Phone:     &phone,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// mockAuthESTransport wraps the ES client helper for auth tests.
func authESRoundTrip(body string) *elasticsearch.Client {
	c, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: rtFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
			}, nil
		}),
	})
	return c
}

// ─── Register ───────────────────────────────────────────────────────────────

func TestAuthUsecase_Register_Success(t *testing.T) {
	f := newAuthUCFix(t, esOK(`{"result":"created"}`))
	ctx := context.Background()

	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", "bob@example.com").Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectCommit()

	// FIXED: Now expecting exactly 168 hours
	f.redisMock.Regexp().ExpectSet(".*", ".*", 168*time.Hour).SetVal("OK")

	res, err := f.uc.Register(ctx, request.RegisterRequest{
		Email: "bob@example.com", Password: "password123", Name: "Bob", Role: "user",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestAuthUsecase_Register_EmailConflict(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", "bob@example.com").Return(true, nil)

	_, err := f.uc.Register(ctx, request.RegisterRequest{
		Email: "bob@example.com", Password: "pass1234", Name: "Bob", Role: "user",
	})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestAuthUsecase_Register_CreateFails(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).
		Return(apperror.Internal("create failed", errors.New("db")))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Register(ctx, request.RegisterRequest{
		Email: "bob@example.com", Password: "pass1234", Name: "Bob", Role: "user",
	})
	require.Error(t, err)
}

func TestAuthUsecase_Register_ESIndexFails(t *testing.T) {
	f := newAuthUCFix(t, esFail())
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectRollback()

	_, err := f.uc.Register(ctx, request.RegisterRequest{
		Email: "bob@example.com", Password: "pass1234", Name: "Bob", Role: "user",
	})
	require.Error(t, err)
}

func TestAuthUsecase_Register_CommitFails(t *testing.T) {
	f := newAuthUCFix(t, esOK(`{"result":"created"}`))
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Register(ctx, request.RegisterRequest{
		Email: "bob@example.com", Password: "pass1234", Name: "Bob", Role: "user",
	})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestAuthUsecase_Login_Success(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()

	const plainPwd = "password123"
	u := newAuthUser(t, "bob@example.com", plainPwd)

	f.repo.EXPECT().FindByEmail(ctx, mock.Anything, "bob@example.com").Return(u, nil)
	f.redisMock.Regexp().ExpectSet(".*", ".*", 168*time.Hour).SetVal("OK")

	res, err := f.uc.Login(ctx, request.LoginRequest{
		Email: "bob@example.com", Password: plainPwd,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
}

func TestAuthUsecase_Login_UserNotFound(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()

	f.repo.EXPECT().FindByEmail(ctx, mock.Anything, "noone@example.com").
		Return(nil, apperror.NotFound("user", "noone@example.com"))

	_, err := f.uc.Login(ctx, request.LoginRequest{Email: "noone@example.com", Password: "pass"})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestAuthUsecase_Login_WrongPassword(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	u := newAuthUser(t, "bob@example.com", "correctpassword")

	f.repo.EXPECT().FindByEmail(ctx, mock.Anything, "bob@example.com").Return(u, nil)

	_, err := f.uc.Login(ctx, request.LoginRequest{Email: "bob@example.com", Password: "wrongpassword"})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestAuthUsecase_Login_RedisStoreFails(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	const plainPwd = "password123"
	u := newAuthUser(t, "bob@example.com", plainPwd)

	f.repo.EXPECT().FindByEmail(ctx, mock.Anything, "bob@example.com").Return(u, nil)
	f.redisMock.Regexp().ExpectSet(".*", ".*", 168*time.Hour).SetErr(errors.New("redis unavailable"))

	_, err := f.uc.Login(ctx, request.LoginRequest{Email: "bob@example.com", Password: plainPwd})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func generateRefreshToken(t *testing.T, jwtMgr *jwtpkg.Manager, userID uuid.UUID) string {
	t.Helper()
	tok, err := jwtMgr.GenerateRefreshToken(userID)
	require.NoError(t, err)
	return tok
}

func TestAuthUsecase_Refresh_Success(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	u := newUser()
	u.Role = "user"
	refreshToken := generateRefreshToken(t, f.jwtMgr, u.ID)

	f.redisMock.ExpectGet(rediskey.AuthRefresh(u.ID)).SetVal(refreshToken)
	f.redisMock.ExpectDel(rediskey.AuthRefresh(u.ID)).SetVal(1)
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)

	f.redisMock.Regexp().ExpectSet(".*", ".*", 168*time.Hour).SetVal("OK")

	res, err := f.uc.Refresh(ctx, request.RefreshRequest{RefreshToken: refreshToken})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestAuthUsecase_Refresh_InvalidToken(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()

	_, err := f.uc.Refresh(ctx, request.RefreshRequest{RefreshToken: "not.a.valid.jwt"})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestAuthUsecase_Refresh_TokenNotInRedis(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	u := newUser()
	refreshToken := generateRefreshToken(t, f.jwtMgr, u.ID)

	f.redisMock.ExpectGet(rediskey.AuthRefresh(u.ID)).RedisNil()

	_, err := f.uc.Refresh(ctx, request.RefreshRequest{RefreshToken: refreshToken})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestAuthUsecase_Refresh_TokenMismatch(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	u := newUser()
	refreshToken := generateRefreshToken(t, f.jwtMgr, u.ID)

	f.redisMock.ExpectGet(rediskey.AuthRefresh(u.ID)).SetVal("different-token-stored")

	_, err := f.uc.Refresh(ctx, request.RefreshRequest{RefreshToken: refreshToken})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestAuthUsecase_Refresh_UserNotFound(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	u := newUser()
	refreshToken := generateRefreshToken(t, f.jwtMgr, u.ID)

	f.redisMock.ExpectGet(rediskey.AuthRefresh(u.ID)).SetVal(refreshToken)
	f.redisMock.ExpectDel(rediskey.AuthRefresh(u.ID)).SetVal(1)
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(nil, apperror.NotFound("user", u.ID.String()))

	_, err := f.uc.Refresh(ctx, request.RefreshRequest{RefreshToken: refreshToken})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestAuthUsecase_Refresh_RedisStoreFails(t *testing.T) {
	f := newAuthUCFix(t, nil)
	ctx := context.Background()
	u := newUser()
	u.Role = "user"
	refreshToken := generateRefreshToken(t, f.jwtMgr, u.ID)

	f.redisMock.ExpectGet(rediskey.AuthRefresh(u.ID)).SetVal(refreshToken)
	f.redisMock.ExpectDel(rediskey.AuthRefresh(u.ID)).SetVal(1)
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)

	f.redisMock.Regexp().ExpectSet(".*", ".*", 168*time.Hour).SetErr(errors.New("redis error"))

	_, err := f.uc.Refresh(ctx, request.RefreshRequest{RefreshToken: refreshToken})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

var _ = time.Now // suppress unused import if auth_test is built alone
