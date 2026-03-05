package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return gormDB, mock
}

func newTestUserRepo(t *testing.T) (UserRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := setupTestDB(t)
	return NewUserRepository(db, zap.NewNop()), mock
}

var userCols = []string{"id", "email", "password", "name", "phone", "role", "created_at", "updated_at", "deleted_at"}

func userRow(u model.User) *sqlmock.Rows {
	return sqlmock.NewRows(userCols).AddRow(
		u.ID, u.Email, u.Password, u.Name, u.Phone, u.Role, u.CreatedAt, u.UpdatedAt, u.DeletedAt,
	)
}

func testUserFixture() model.User {
	return model.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Password:  "hashedpw",
		Name:      "Test User",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestUserRepository_Create_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectExec(`INSERT INTO "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), nil, &u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectExec(`INSERT INTO "users"`).WillReturnError(errors.New("db error"))

	err := repo.Create(context.Background(), nil, &u)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserRepository_Update_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectExec(`UPDATE "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Update(context.Background(), nil, &u)
	assert.NoError(t, err)
}

func TestUserRepository_Update_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectExec(`UPDATE "users"`).WillReturnError(errors.New("db error"))

	err := repo.Update(context.Background(), nil, &u)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// SoftDelete
// ---------------------------------------------------------------------------

func TestUserRepository_SoftDelete_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectExec(`UPDATE "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.SoftDelete(context.Background(), nil, id)
	assert.NoError(t, err)
}

func TestUserRepository_SoftDelete_NotFound(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectExec(`UPDATE "users"`).
		WillReturnResult(sqlmock.NewResult(1, 0)) // 0 rows affected

	err := repo.SoftDelete(context.Background(), nil, id)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUserRepository_SoftDelete_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectExec(`UPDATE "users"`).WillReturnError(errors.New("db error"))

	err := repo.SoftDelete(context.Background(), nil, id)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// HardDelete
// ---------------------------------------------------------------------------

func TestUserRepository_HardDelete_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.HardDelete(context.Background(), nil, id)
	assert.NoError(t, err)
}

func TestUserRepository_HardDelete_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM "users"`).WillReturnError(errors.New("db error"))

	err := repo.HardDelete(context.Background(), nil, id)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// FindByID
// ---------------------------------------------------------------------------

func TestUserRepository_FindByID_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).
		WillReturnRows(userRow(u))

	result, err := repo.FindByID(context.Background(), nil, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, result.ID)
	assert.Equal(t, u.Email, result.Email)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).
		WillReturnRows(sqlmock.NewRows(userCols)) // empty

	result, err := repo.FindByID(context.Background(), nil, id)
	assert.Nil(t, result)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUserRepository_FindByID_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).WillReturnError(errors.New("db error"))

	result, err := repo.FindByID(context.Background(), nil, id)
	assert.Nil(t, result)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// FindAll
// ---------------------------------------------------------------------------

func TestUserRepository_FindAll_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()
	f := request.UserFilter{}
	f.Page = 1
	f.PageSize = 20

	// COUNT query
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// DATA query
	mock.ExpectQuery(`SELECT .* FROM "users"`).
		WillReturnRows(userRow(u))

	users, total, err := repo.FindAll(context.Background(), nil, f)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
}

func TestUserRepository_FindAll_Empty(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	f := request.UserFilter{}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .* FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols))

	users, total, err := repo.FindAll(context.Background(), nil, f)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, users)
}

func TestUserRepository_FindAll_CountError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	f := request.UserFilter{}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).WillReturnError(errors.New("db error"))

	_, _, err := repo.FindAll(context.Background(), nil, f)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

func TestUserRepository_FindAll_DataError(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	f := request.UserFilter{}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))

	_, _, err := repo.FindAll(context.Background(), nil, f)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// Count
// ---------------------------------------------------------------------------

func TestUserRepository_Count_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.Count(context.Background(), nil, request.UserFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestUserRepository_Count_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).WillReturnError(errors.New("db error"))

	_, err := repo.Count(context.Background(), nil, request.UserFilter{})
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// Exists
// ---------------------------------------------------------------------------

func TestUserRepository_Exists_True(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.Exists(context.Background(), nil, "email", "test@example.com")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepository_Exists_False(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	exists, err := repo.Exists(context.Background(), nil, "email", "no@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserRepository_Exists_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).WillReturnError(errors.New("db error"))

	_, err := repo.Exists(context.Background(), nil, "email", "test@example.com")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// FindByEmail
// ---------------------------------------------------------------------------

func TestUserRepository_FindByEmail_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).
		WillReturnRows(userRow(u))

	result, err := repo.FindByEmail(context.Background(), nil, u.Email)
	require.NoError(t, err)
	assert.Equal(t, u.Email, result.Email)
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).
		WillReturnRows(sqlmock.NewRows(userCols))

	result, err := repo.FindByEmail(context.Background(), nil, "no@example.com")
	assert.Nil(t, result)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUserRepository_FindByEmail_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).WillReturnError(errors.New("db error"))

	_, err := repo.FindByEmail(context.Background(), nil, "test@example.com")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}

// ---------------------------------------------------------------------------
// FindByIDs
// ---------------------------------------------------------------------------

func TestUserRepository_FindByIDs_Success(t *testing.T) {
	repo, mock := newTestUserRepo(t)
	u := testUserFixture()

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).
		WillReturnRows(userRow(u))

	results, err := repo.FindByIDs(context.Background(), nil, []uuid.UUID{u.ID})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestUserRepository_FindByIDs_Empty(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).
		WillReturnRows(sqlmock.NewRows(userCols))

	results, err := repo.FindByIDs(context.Background(), nil, []uuid.UUID{uuid.New()})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestUserRepository_FindByIDs_DBError(t *testing.T) {
	repo, mock := newTestUserRepo(t)

	mock.ExpectQuery(`SELECT .* FROM "users" WHERE`).WillReturnError(errors.New("db error"))

	_, err := repo.FindByIDs(context.Background(), nil, []uuid.UUID{uuid.New()})
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.CodeInternal, appErr.Code)
}
