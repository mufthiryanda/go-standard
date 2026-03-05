package usecase

import (
	"context"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"go-standard/internal/apperror"
	"go-standard/internal/config"
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	"go-standard/internal/pkg/storage"
)

// MediaUsecase defines all file-related business operations.
type MediaUsecase interface {
	// UploadAvatar uploads a user avatar and returns the result.
	// Public flag is set to true — the URL is returned directly.
	UploadAvatar(ctx context.Context, userID uuid.UUID, fh *multipart.FileHeader) (*response.MediaUploadResponse, error)

	// GenerateUploadURL creates a presigned PUT URL so the client can upload
	// a file directly to object storage. Returns the URL and the pre-generated key.
	GenerateUploadURL(ctx context.Context, req request.GenerateUploadURLRequest) (*response.PresignedURLResponse, error)

	// GetDownloadURL creates a time-limited presigned GET URL for a private object.
	GetDownloadURL(ctx context.Context, key string) (string, error)

	// DeleteFile removes an object from the default bucket.
	DeleteFile(ctx context.Context, key string) error
}

type mediaUsecase struct {
	storage storage.StorageManager
	cfg     *config.Config
	logger  *zap.Logger
}

// NewMediaUsecase constructs a MediaUsecase with the supplied dependencies.
func NewMediaUsecase(
	storage storage.StorageManager,
	cfg *config.Config,
	logger *zap.Logger,
) MediaUsecase {
	return &mediaUsecase{
		storage: storage,
		cfg:     cfg,
		logger:  logger,
	}
}

// UploadAvatar validates that the uploaded file is an image, then delegates to
// StorageManager. The object is made public; PublicURL is returned to callers.
func (u *mediaUsecase) UploadAvatar(
	ctx context.Context,
	userID uuid.UUID,
	fh *multipart.FileHeader,
) (*response.MediaUploadResponse, error) {
	contentType := fh.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, apperror.BadRequest("avatar must be an image file")
	}

	f, err := fh.Open()
	if err != nil {
		u.logger.Error("media: failed to open uploaded file",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, apperror.BadRequest("cannot open uploaded file")
	}
	defer f.Close()

	result, err := u.storage.Upload(ctx, storage.UploadInput{
		Entity:      "users/avatars",
		Filename:    fh.Filename,
		Content:     f,
		ContentType: contentType,
		Size:        fh.Size,
		Public:      true,
	})
	if err != nil {
		u.logger.Error("media: avatar upload failed",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return &response.MediaUploadResponse{
		Key:       result.Key,
		PublicURL: result.PublicURL,
	}, nil
}

// GenerateUploadURL creates a presigned PUT URL so the client can push a file
// directly to object storage without routing through the application server.
// The key is generated here so the caller knows where the object will live.
func (u *mediaUsecase) GenerateUploadURL(
	ctx context.Context,
	req request.GenerateUploadURLRequest,
) (*response.PresignedURLResponse, error) {
	key := storage.GenerateKey(req.Entity, req.Filename)
	ttl := time.Duration(u.cfg.Storage.PresignedPutTTL) * time.Second

	uploadURL, err := u.storage.PresignPutURL(ctx, u.cfg.Storage.DefaultBucket, key, ttl)
	if err != nil {
		u.logger.Error("media: failed to generate presigned PUT URL",
			zap.String("entity", req.Entity),
			zap.Error(err),
		)
		return nil, err
	}

	return &response.PresignedURLResponse{
		UploadURL: uploadURL,
		Key:       key,
		ExpiresIn: u.cfg.Storage.PresignedPutTTL,
	}, nil
}

// GetDownloadURL returns a time-limited presigned GET URL for a private object.
func (u *mediaUsecase) GetDownloadURL(ctx context.Context, key string) (string, error) {
	ttl := time.Duration(u.cfg.Storage.PresignedGetTTL) * time.Second

	url, err := u.storage.PresignGetURL(ctx, u.cfg.Storage.DefaultBucket, key, ttl)
	if err != nil {
		u.logger.Error("media: failed to generate presigned GET URL",
			zap.String("key", key),
			zap.Error(err),
		)
		return "", err
	}
	return url, nil
}

// DeleteFile removes an object from the default bucket.
func (u *mediaUsecase) DeleteFile(ctx context.Context, key string) error {
	if err := u.storage.Delete(ctx, u.cfg.Storage.DefaultBucket, key); err != nil {
		u.logger.Error("media: failed to delete file",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}
