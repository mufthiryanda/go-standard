package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
)

// UploadMultipart uploads a file using the S3 multipart upload protocol.
// s3manager.Uploader automatically splits the upload into parts when
// the body exceeds PartSize.
func (a *s3adapter) UploadMultipart(ctx context.Context, input MultipartUploadInput) (*UploadResult, error) {
	if err := ValidateFile(input.Filename, input.ContentType, input.Size, a.cfg); err != nil {
		return nil, err
	}

	key := GenerateKey(input.Entity, input.Filename)
	bucket := resolveBucket(input.Bucket, a.cfg.DefaultBucket)

	partSize := input.PartSizeBytes
	if partSize == 0 {
		partSize = 5 * 1024 * 1024 // 5 MB
	}
	concurrency := input.Concurrency
	if concurrency == 0 {
		concurrency = 3
	}

	upInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        input.Content,
		ContentType: aws.String(input.ContentType),
	}
	if input.Public {
		upInput.ACL = s3types.ObjectCannedACLPublicRead
	}
	if len(input.Metadata) > 0 {
		upInput.Metadata = input.Metadata
	}

	out, err := a.uploader.Upload(ctx, upInput,
		func(u *s3manager.Uploader) {
			u.PartSize = partSize
			u.Concurrency = concurrency
		},
	)
	if err != nil {
		a.logger.Error("storage: multipart upload failed",
			zap.String("provider", string(a.provider)),
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, MapS3Error(err, string(a.provider))
	}

	result := &UploadResult{
		Key:    key,
		Bucket: bucket,
		ETag:   aws.ToString(out.ETag),
		Size:   input.Size,
	}
	if input.Public {
		result.PublicURL = a.PublicURL(key)
	}
	return result, nil
}
