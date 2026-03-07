package storage

import (
	"context"
	"io"
	"time"
)

// ProviderType identifies a storage backend.
type ProviderType string

const (
	ProviderS3       ProviderType = "s3"
	ProviderMinIO    ProviderType = "minio"
	ProviderDOSpaces ProviderType = "do_spaces"
)

// UploadInput holds all parameters for a single-part upload.
type UploadInput struct {
	Bucket      string // empty = use default from config
	Entity      string // e.g. "users/avatars" — prefix for key generation
	Filename    string // original filename, used for extension extraction
	Content     io.Reader
	ContentType string            // MIME type — validated against whitelist
	Size        int64             // validated against MaxFileSizeBytes
	Public      bool              // true → ObjectCannedACLPublicRead
	Metadata    map[string]string // stored as S3 object metadata
}

// MultipartUploadInput extends UploadInput with multipart-specific settings.
type MultipartUploadInput struct {
	UploadInput
	PartSizeBytes int64 // default: 5MB
	Concurrency   int   // default: 3 parallel part uploads
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	Key       string // generated key: {entity}/{uuid}.{ext}
	Bucket    string
	ETag      string
	Size      int64
	PublicURL string // populated only when Public == true
}

// DownloadResult wraps the downloaded object body and metadata.
// Caller is responsible for closing Body.
type DownloadResult struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
	ETag        string
	Metadata    map[string]string
}

// CopyInput specifies source and destination for an object copy.
type CopyInput struct {
	SrcBucket string
	SrcKey    string
	DstBucket string // empty = same as SrcBucket
	DstKey    string
	Public    bool
}

// ListInput holds listing parameters with pagination support.
type ListInput struct {
	Bucket            string
	Prefix            string
	MaxKeys           int32  // default: 100, max: 1000
	ContinuationToken string // pagination cursor
}

// ListResult holds the listing response.
type ListResult struct {
	Objects               []ObjectInfo
	NextContinuationToken string
	IsTruncated           bool
}

// ObjectInfo describes a single stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}

// StorageClient is the per-provider interface.
// Consumed by StorageManager — callers always go through StorageManager.
type StorageClient interface {
	Upload(ctx context.Context, input UploadInput) (*UploadResult, error)
	UploadMultipart(ctx context.Context, input MultipartUploadInput) (*UploadResult, error)
	Download(ctx context.Context, bucket, key string) (*DownloadResult, error)
	Delete(ctx context.Context, bucket, key string) error
	Copy(ctx context.Context, input CopyInput) error
	List(ctx context.Context, input ListInput) (*ListResult, error)
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PresignPutURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PublicURL(key string) string
	Provider() ProviderType
}

// StorageManager is what gets injected into usecases via Wire.
// Switch changes the active provider atomically at runtime without restart.
type StorageManager interface {
	StorageClient

	// Switch changes the active provider at runtime without restart.
	// Returns apperror.BadRequest if the provider is not registered.
	Switch(provider ProviderType) error

	// ActiveProvider returns the currently active provider type.
	ActiveProvider() ProviderType
}
