package storage

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"go-standard/internal/apperror"
	"go-standard/internal/config"
)

type storageManager struct {
	clients map[ProviderType]StorageClient
	active  atomic.Value // stores string (ProviderType)
	cfg     config.StorageConfig
	logger  *zap.Logger
}

// NewStorageManager initialises all configured storage providers and returns
// a StorageManager that delegates to the active provider.
func NewStorageManager(cfg config.StorageConfig, logger *zap.Logger) (StorageManager, func(), error) {
	m := &storageManager{
		clients: make(map[ProviderType]StorageClient),
		cfg:     cfg,
		logger:  logger,
	}

	s3a, err := newS3Adapter(cfg, logger)
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("storage: s3 init failed", err)
	}
	m.clients[ProviderS3] = s3a

	minio, err := newMinIOAdapter(cfg, logger)
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("storage: minio init failed", err)
	}
	m.clients[ProviderMinIO] = minio

	dos, err := newDOSpacesAdapter(cfg, logger)
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("storage: do_spaces init failed", err)
	}
	m.clients[ProviderDOSpaces] = dos

	active := ProviderType(cfg.ActiveProvider)
	if _, ok := m.clients[active]; !ok {
		return nil, nil, apperror.Internal(
			fmt.Sprintf("storage: active provider %q not registered", active), nil,
		)
	}
	m.active.Store(string(active))

	logger.Info("storage: manager initialised",
		zap.String("active_provider", string(active)),
	)

	cleanup := func() {
		logger.Info("storage: manager released")
	}
	return m, cleanup, nil
}

func (m *storageManager) Switch(provider ProviderType) error {
	if _, ok := m.clients[provider]; !ok {
		return apperror.BadRequest(fmt.Sprintf("storage: provider %q not registered", provider))
	}
	m.active.Store(string(provider))
	m.logger.Info("storage: provider switched", zap.String("provider", string(provider)))
	return nil
}

func (m *storageManager) ActiveProvider() ProviderType {
	return ProviderType(m.active.Load().(string))
}

func (m *storageManager) activeClient() StorageClient {
	return m.clients[m.ActiveProvider()]
}

// --- StorageClient delegation ---

func (m *storageManager) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	return m.activeClient().Upload(ctx, input)
}

func (m *storageManager) UploadMultipart(ctx context.Context, input MultipartUploadInput) (*UploadResult, error) {
	return m.activeClient().UploadMultipart(ctx, input)
}

func (m *storageManager) Download(ctx context.Context, bucket, key string) (*DownloadResult, error) {
	return m.activeClient().Download(ctx, bucket, key)
}

func (m *storageManager) Delete(ctx context.Context, bucket, key string) error {
	return m.activeClient().Delete(ctx, bucket, key)
}

func (m *storageManager) Copy(ctx context.Context, input CopyInput) error {
	return m.activeClient().Copy(ctx, input)
}

func (m *storageManager) List(ctx context.Context, input ListInput) (*ListResult, error) {
	return m.activeClient().List(ctx, input)
}

func (m *storageManager) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return m.activeClient().PresignGetURL(ctx, bucket, key, ttl)
}

func (m *storageManager) PresignPutURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return m.activeClient().PresignPutURL(ctx, bucket, key, ttl)
}

func (m *storageManager) PublicURL(key string) string {
	return m.activeClient().PublicURL(key)
}

func (m *storageManager) Provider() ProviderType {
	return m.ActiveProvider()
}
