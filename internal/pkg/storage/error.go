package storage

import (
	"context"
	"errors"
	"fmt"

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"

	"go-standard/internal/apperror"
)

// MapS3Error maps AWS SDK errors to domain AppError types.
func MapS3Error(err error, provider string) error {
	if err == nil {
		return nil
	}

	var noSuchKey *s3types.NoSuchKey
	var noSuchBucket *s3types.NoSuchBucket
	var notFound *s3types.NotFound

	switch {
	case errors.As(err, &noSuchKey), errors.As(err, &notFound):
		return apperror.NotFound(provider, "object")
	case errors.As(err, &noSuchBucket):
		return apperror.NotFound(provider, "bucket")
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "AccessDenied", "Forbidden":
			return apperror.Forbidden(fmt.Sprintf("%s: access denied", provider))
		case "InvalidAccessKeyId", "SignatureDoesNotMatch":
			return apperror.Unauthorized(fmt.Sprintf("%s: invalid credentials", provider))
		case "SlowDown", "ServiceUnavailable":
			return apperror.ServiceUnavailable(fmt.Sprintf("%s: throttled", provider), err)
		}
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return apperror.ServiceUnavailable(fmt.Sprintf("%s: timeout", provider), err)
	}

	return apperror.Internal(fmt.Sprintf("%s: storage operation failed", provider), err)
}
