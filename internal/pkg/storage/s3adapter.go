package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"

	"go-standard/internal/config"
)

type s3adapter struct {
	client    *s3.Client
	uploader  *s3manager.Uploader
	presigner *s3.PresignClient
	cfg       config.StorageConfig
	provider  ProviderType
	logger    *zap.Logger
}

// newS3Adapter constructs an adapter for AWS S3.
func newS3Adapter(cfg config.StorageConfig, logger *zap.Logger) (*s3adapter, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Providers.S3.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.Providers.S3.AccessKeyID,
			cfg.Providers.S3.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg)
	return buildAdapter(client, cfg, ProviderS3, logger), nil
}

// newMinIOAdapter constructs an adapter for MinIO.
func newMinIOAdapter(cfg config.StorageConfig, logger *zap.Logger) (*s3adapter, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.Providers.MinIO.AccessKeyID,
			cfg.Providers.MinIO.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}
	awsCfg.BaseEndpoint = aws.String(cfg.Providers.MinIO.Endpoint)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return buildAdapter(client, cfg, ProviderMinIO, logger), nil
}

// newDOSpacesAdapter constructs an adapter for DigitalOcean Spaces.
func newDOSpacesAdapter(cfg config.StorageConfig, logger *zap.Logger) (*s3adapter, error) {
	endpoint := fmt.Sprintf("https://%s.digitaloceanspaces.com", cfg.Providers.DOSpaces.Region)
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Providers.DOSpaces.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.Providers.DOSpaces.AccessKeyID,
			cfg.Providers.DOSpaces.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}
	awsCfg.BaseEndpoint = aws.String(endpoint)
	client := s3.NewFromConfig(awsCfg)
	return buildAdapter(client, cfg, ProviderDOSpaces, logger), nil
}

func buildAdapter(client *s3.Client, cfg config.StorageConfig, provider ProviderType, logger *zap.Logger) *s3adapter {
	return &s3adapter{
		client:    client,
		uploader:  s3manager.NewUploader(client),
		presigner: s3.NewPresignClient(client),
		cfg:       cfg,
		provider:  provider,
		logger:    logger,
	}
}

func resolveBucket(input, defaultBucket string) string {
	if input != "" {
		return input
	}
	return defaultBucket
}

func (a *s3adapter) Provider() ProviderType {
	return a.provider
}

func (a *s3adapter) PublicURL(key string) string {
	base := strings.TrimRight(a.cfg.PublicBaseURL, "/")
	return fmt.Sprintf("%s/%s", base, key)
}

func (a *s3adapter) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	if err := ValidateFile(input.Filename, input.ContentType, input.Size, a.cfg); err != nil {
		return nil, err
	}

	key := GenerateKey(input.Entity, input.Filename)
	bucket := resolveBucket(input.Bucket, a.cfg.DefaultBucket)

	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        input.Content,
		ContentType: aws.String(input.ContentType),
	}
	if input.Public {
		putInput.ACL = s3types.ObjectCannedACLPublicRead
	}
	if len(input.Metadata) > 0 {
		putInput.Metadata = input.Metadata
	}

	out, err := a.client.PutObject(ctx, putInput)
	if err != nil {
		a.logger.Error("storage: upload failed",
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

func (a *s3adapter) Download(ctx context.Context, bucket, key string) (*DownloadResult, error) {
	bucket = resolveBucket(bucket, a.cfg.DefaultBucket)

	out, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, MapS3Error(err, string(a.provider))
	}

	result := &DownloadResult{
		Body:     out.Body,
		ETag:     aws.ToString(out.ETag),
		Metadata: out.Metadata,
	}
	if out.ContentType != nil {
		result.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		result.Size = *out.ContentLength
	}
	return result, nil
}

func (a *s3adapter) Delete(ctx context.Context, bucket, key string) error {
	bucket = resolveBucket(bucket, a.cfg.DefaultBucket)
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return MapS3Error(err, string(a.provider))
	}
	return nil
}

func (a *s3adapter) Copy(ctx context.Context, input CopyInput) error {
	srcBucket := resolveBucket(input.SrcBucket, a.cfg.DefaultBucket)
	dstBucket := resolveBucket(input.DstBucket, srcBucket)
	copySource := fmt.Sprintf("%s/%s", srcBucket, input.SrcKey)

	copyInput := &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(input.DstKey),
		CopySource: aws.String(copySource),
	}
	if input.Public {
		copyInput.ACL = s3types.ObjectCannedACLPublicRead
	}

	_, err := a.client.CopyObject(ctx, copyInput)
	if err != nil {
		return MapS3Error(err, string(a.provider))
	}
	return nil
}

func (a *s3adapter) List(ctx context.Context, input ListInput) (*ListResult, error) {
	bucket := resolveBucket(input.Bucket, a.cfg.DefaultBucket)
	maxKeys := input.MaxKeys
	if maxKeys == 0 {
		maxKeys = 100
	}

	listInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: &maxKeys,
	}
	if input.Prefix != "" {
		listInput.Prefix = aws.String(input.Prefix)
	}
	if input.ContinuationToken != "" {
		listInput.ContinuationToken = aws.String(input.ContinuationToken)
	}

	out, err := a.client.ListObjectsV2(ctx, listInput)
	if err != nil {
		return nil, MapS3Error(err, string(a.provider))
	}

	objects := make([]ObjectInfo, 0, len(out.Contents))
	for _, obj := range out.Contents {
		info := ObjectInfo{
			Key:  aws.ToString(obj.Key),
			Size: aws.ToInt64(obj.Size),
			ETag: aws.ToString(obj.ETag),
		}
		if obj.LastModified != nil {
			info.LastModified = obj.LastModified.In(jakartaLoc())
		}
		objects = append(objects, info)
	}

	result := &ListResult{
		Objects:     objects,
		IsTruncated: aws.ToBool(out.IsTruncated),
	}
	if out.NextContinuationToken != nil {
		result.NextContinuationToken = *out.NextContinuationToken
	}
	return result, nil
}

func (a *s3adapter) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	bucket = resolveBucket(bucket, a.cfg.DefaultBucket)
	out, err := a.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", MapS3Error(err, string(a.provider))
	}
	return out.URL, nil
}

func (a *s3adapter) PresignPutURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	bucket = resolveBucket(bucket, a.cfg.DefaultBucket)
	out, err := a.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", MapS3Error(err, string(a.provider))
	}
	return out.URL, nil
}

// jakartaLoc returns the Asia/Jakarta timezone, falling back to UTC on error.
func jakartaLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.UTC
	}
	return loc
}

// UploadMultipart is implemented in multipart.go.
