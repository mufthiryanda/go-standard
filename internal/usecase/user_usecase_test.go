package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/config"
	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ─── ES helpers ─────────────────────────────────────────────────────────────

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func esOK(body string) *elasticsearch.Client {
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

func esFail() *elasticsearch.Client {
	c, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: rtFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error":"es"}`)),
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
			}, nil
		}),
	})
	return c
}

func esNet() *elasticsearch.Client {
	c, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: rtFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		}),
	})
	return c
}

// ─── JWT helper ─────────────────────────────────────────────────────────────

func newJWTMgr(t *testing.T) *jwtpkg.Manager {
	t.Helper()
	cfg := &config.Config{}
	cfg.JWT.Secret = "super-secret-test-key-minimum-32chars!!"
	cfg.JWT.AccessTTL = "15m"
	cfg.JWT.RefreshTTL = "168h"
	cfg.JWT.Issuer = "test"
	m, err := jwtpkg.NewManager(cfg)
	require.NoError(t, err)
	return m
}

// ─── Fixture ────────────────────────────────────────────────────────────────

type ucFix struct {
	uc        UserUsecase
	repo      *mocks.MockUserRepository
	dbMock    sqlmock.Sqlmock
	redisMock redisMockPkg.ClientMock
}

func newUCFix(t *testing.T, es *elasticsearch.Client) *ucFix {
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
	uc := NewUserUsecase(gDB, repo, rdb, es, newJWTMgr(t), zap.NewNop())

	return &ucFix{uc: uc, repo: repo, dbMock: dbMock, redisMock: rMock}
}

func newUser() *model.User {
	phone := "123-456-7890" // Provide dummy phone to prevent mapping panics
	return &model.User{
		ID: uuid.New(), Email: "alice@example.com",
		Password: "hashedpw", Name: "Alice", Role: "user", Phone: &phone,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// ─── Register ───────────────────────────────────────────────────────────────

func TestUserUsecase_Register_Success(t *testing.T) {
	f := newUCFix(t, esOK(`{"result":"created"}`))
	ctx := context.Background()

	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", "alice@example.com").Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectCommit()

	res, err := f.uc.Register(ctx, request.CreateUserRequest{
		Email: "alice@example.com", Password: "password123", Name: "Alice", Role: "user",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", res.Email)
}

func TestUserUsecase_Register_EmailConflict(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", "alice@example.com").Return(true, nil)

	_, err := f.uc.Register(ctx, request.CreateUserRequest{
		Email: "alice@example.com", Password: "pass1234", Name: "Alice", Role: "user",
	})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestUserUsecase_Register_ExistsDBError(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).
		Return(false, apperror.Internal("db", errors.New("conn")))

	_, err := f.uc.Register(ctx, request.CreateUserRequest{
		Email: "alice@example.com", Password: "pass1234", Name: "Alice", Role: "user",
	})
	require.Error(t, err)
}

func TestUserUsecase_Register_CreateFails(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).
		Return(apperror.Internal("create failed", errors.New("db")))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Register(ctx, request.CreateUserRequest{
		Email: "alice@example.com", Password: "pass1234", Name: "Alice", Role: "user",
	})
	require.Error(t, err)
}

func TestUserUsecase_Register_ESIndexFails(t *testing.T) {
	f := newUCFix(t, esFail())
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectRollback()

	_, err := f.uc.Register(ctx, request.CreateUserRequest{
		Email: "alice@example.com", Password: "pass1234", Name: "Alice", Role: "user",
	})
	require.Error(t, err)
}

func TestUserUsecase_Register_CommitFails(t *testing.T) {
	f := newUCFix(t, esOK(`{"result":"created"}`))
	ctx := context.Background()
	f.repo.EXPECT().Exists(ctx, mock.Anything, "email", mock.Anything).Return(false, nil)
	f.dbMock.ExpectBegin()
	f.repo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Register(ctx, request.CreateUserRequest{
		Email: "alice@example.com", Password: "pass1234", Name: "Alice", Role: "user",
	})
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestUserUsecase_GetByID_CacheHit(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	u := newUser()

	resp := response.UserResponse{ID: u.ID.String(), Email: u.Email, Name: u.Name, Role: u.Role}
	b, _ := json.Marshal(resp)
	f.redisMock.ExpectGet(rediskey.UserDetail(u.ID)).SetVal(string(b))

	res, err := f.uc.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), res.ID)
}

func TestUserUsecase_GetByID_CacheMiss_Found(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	u := newUser()

	f.redisMock.ExpectGet(rediskey.UserDetail(u.ID)).RedisNil()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)

	res, err := f.uc.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.Email, res.Email)
}

func TestUserUsecase_GetByID_CacheMiss_NotFound(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	id := uuid.New()

	f.redisMock.ExpectGet(rediskey.UserDetail(id)).RedisNil()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, id).Return(nil, apperror.NotFound("user", id.String()))

	_, err := f.uc.GetByID(ctx, id)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

// ─── GetAll ──────────────────────────────────────────────────────────────────

func TestUserUsecase_GetAll_NoKeyword_Success(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	u := newUser()
	filter := request.UserFilter{}
	filter.Page = 1
	filter.PageSize = 20

	f.repo.EXPECT().FindAll(ctx, mock.Anything, filter).Return([]model.User{*u}, int64(1), nil)

	users, meta, err := f.uc.GetAll(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, int64(1), meta.TotalItems)
}

func TestUserUsecase_GetAll_NoKeyword_RepoError(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	filter := request.UserFilter{}

	f.repo.EXPECT().FindAll(ctx, mock.Anything, filter).
		Return(nil, int64(0), apperror.Internal("db", errors.New("db")))

	_, _, err := f.uc.GetAll(ctx, filter)
	require.Error(t, err)
}

func TestUserUsecase_GetAll_Keyword_Success(t *testing.T) {
	u := newUser()
	esBody := `{"hits":{"total":{"value":1,"relation":"eq"},"hits":[{"_id":"` + u.ID.String() + `"}]}}`
	f := newUCFix(t, esOK(esBody))
	ctx := context.Background()
	kw := "alice"
	filter := request.UserFilter{}
	filter.Keyword = &kw
	filter.Page = 1
	filter.PageSize = 20

	f.repo.EXPECT().FindByIDs(ctx, mock.Anything, mock.Anything).Return([]model.User{*u}, nil)

	users, meta, err := f.uc.GetAll(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, int64(1), meta.TotalItems)
}

func TestUserUsecase_GetAll_Keyword_ESNetworkError(t *testing.T) {
	f := newUCFix(t, esNet())
	ctx := context.Background()
	kw := "alice"
	filter := request.UserFilter{}
	filter.Keyword = &kw

	_, _, err := f.uc.GetAll(ctx, filter)
	require.Error(t, err)
}

func TestUserUsecase_GetAll_Keyword_EmptyResults(t *testing.T) {
	esBody := `{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`
	f := newUCFix(t, esOK(esBody))
	ctx := context.Background()
	kw := "nobody"
	filter := request.UserFilter{}
	filter.Keyword = &kw

	users, meta, err := f.uc.GetAll(ctx, filter)
	require.NoError(t, err)
	assert.Empty(t, users)
	assert.Equal(t, int64(0), meta.TotalItems)
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestUserUsecase_Update_Success(t *testing.T) {
	f := newUCFix(t, esOK(`{"result":"updated"}`))
	ctx := context.Background()
	u := newUser()
	newName := "Alice Updated"

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)
	f.repo.EXPECT().Update(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectCommit()

	res, err := f.uc.Update(ctx, u.ID, request.UpdateUserRequest{Name: &newName}, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, newName, res.Name)
}

func TestUserUsecase_Update_UserNotFound(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	id := uuid.New()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, id).Return(nil, apperror.NotFound("user", id.String()))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Update(ctx, id, request.UpdateUserRequest{}, uuid.New())
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUserUsecase_Update_RepoUpdateError(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	u := newUser()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)
	f.repo.EXPECT().Update(ctx, mock.Anything, mock.Anything).
		Return(apperror.Internal("update failed", errors.New("db")))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Update(ctx, u.ID, request.UpdateUserRequest{}, uuid.New())
	require.Error(t, err)
}

func TestUserUsecase_Update_ESReindexError(t *testing.T) {
	f := newUCFix(t, esFail())
	ctx := context.Background()
	u := newUser()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)
	f.repo.EXPECT().Update(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectRollback()

	_, err := f.uc.Update(ctx, u.ID, request.UpdateUserRequest{}, uuid.New())
	require.Error(t, err)
}

func TestUserUsecase_Update_CommitFails(t *testing.T) {
	f := newUCFix(t, esOK(`{"result":"updated"}`))
	ctx := context.Background()
	u := newUser()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().FindByID(ctx, mock.Anything, u.ID).Return(u, nil)
	f.repo.EXPECT().Update(ctx, mock.Anything, mock.Anything).Return(nil)
	f.dbMock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	f.dbMock.ExpectRollback()

	_, err := f.uc.Update(ctx, u.ID, request.UpdateUserRequest{}, uuid.New())
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ─── Delete ──────────────────────────────────────────────────────────────────

func TestUserUsecase_Delete_Success(t *testing.T) {
	f := newUCFix(t, esOK(`{"result":"deleted"}`))
	ctx := context.Background()
	id := uuid.New()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().SoftDelete(ctx, mock.Anything, id).Return(nil)
	f.dbMock.ExpectCommit()

	require.NoError(t, f.uc.Delete(ctx, id, uuid.New()))
}

func TestUserUsecase_Delete_SoftDeleteNotFound(t *testing.T) {
	f := newUCFix(t, nil)
	ctx := context.Background()
	id := uuid.New()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().SoftDelete(ctx, mock.Anything, id).Return(apperror.NotFound("user", id.String()))
	f.dbMock.ExpectRollback()

	err := f.uc.Delete(ctx, id, uuid.New())
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUserUsecase_Delete_ESDeleteError(t *testing.T) {
	f := newUCFix(t, esFail())
	ctx := context.Background()
	id := uuid.New()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().SoftDelete(ctx, mock.Anything, id).Return(nil)
	f.dbMock.ExpectRollback()

	require.Error(t, f.uc.Delete(ctx, id, uuid.New()))
}

func TestUserUsecase_Delete_CommitFails(t *testing.T) {
	f := newUCFix(t, esOK(`{"result":"deleted"}`))
	ctx := context.Background()
	id := uuid.New()

	f.dbMock.ExpectBegin()
	f.repo.EXPECT().SoftDelete(ctx, mock.Anything, id).Return(nil)
	f.dbMock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	f.dbMock.ExpectRollback()

	err := f.uc.Delete(ctx, id, uuid.New())
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}
