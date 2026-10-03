package storage

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nick130920/fintech-backend/configs"
)

var (
	// ErrInvalidObjectStoreConfig indicates object-storage configuration is incomplete or incoherent.
	ErrInvalidObjectStoreConfig = errors.New("invalid object storage configuration")
	// ErrInvalidPresignRequest indicates the caller provided an invalid presigning request.
	ErrInvalidPresignRequest = errors.New("invalid presign request")
	// ErrObjectStorePresignFailed indicates the object-storage provider could not create a presigned request.
	ErrObjectStorePresignFailed = errors.New("object storage presign failed")
)

// s3Presigner is the narrow AWS SDK seam required to presign object operations.
type s3Presigner interface {
	PresignPutObject(context.Context, *s3.PutObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignGetObject(context.Context, *s3.GetObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// S3ObjectStore creates S3-compatible presigned object URLs.
type S3ObjectStore struct {
	bucket    string
	presigner s3Presigner
}

// NewS3ObjectStore creates an S3-compatible object store using only the static credentials in config.
func NewS3ObjectStore(config configs.UploadConfig) (*S3ObjectStore, error) {
	endpoint, region, bucket, accessKeyID, secretAccessKey, ok := validObjectStoreConfig(config)
	if !ok {
		return nil, ErrInvalidObjectStoreConfig
	}

	client := s3.NewFromConfig(aws.Config{
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		BaseEndpoint: aws.String(endpoint),
	}, func(options *s3.Options) {
		options.UsePathStyle = config.ObjectStorageForcePathStyle
	})

	return newS3ObjectStoreWithPresigner(bucket, s3.NewPresignClient(client)), nil
}

func newS3ObjectStoreWithPresigner(bucket string, presigner s3Presigner) *S3ObjectStore {
	return &S3ObjectStore{bucket: bucket, presigner: presigner}
}

// PresignUpload creates a PUT authorization for one object upload.
func (s *S3ObjectStore) PresignUpload(ctx context.Context, request PresignUploadRequest) (PresignedUpload, error) {
	if !validUploadRequest(request) || s == nil || s.presigner == nil || s.bucket == "" {
		return PresignedUpload{}, ErrInvalidPresignRequest
	}

	result, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(request.Key),
		ContentType:   aws.String(request.ContentType),
		ContentLength: aws.Int64(request.ContentLength),
	}, func(options *s3.PresignOptions) {
		options.Expires = request.ExpiresIn
	})
	if err != nil || result == nil || result.URL == "" {
		return PresignedUpload{}, ErrObjectStorePresignFailed
	}

	headers, ok := singleValueHeaders(result.SignedHeader)
	if !ok {
		return PresignedUpload{}, ErrObjectStorePresignFailed
	}
	return PresignedUpload{URL: result.URL, Headers: headers}, nil
}

// PresignDownload creates a GET authorization for one object download.
func (s *S3ObjectStore) PresignDownload(ctx context.Context, request PresignDownloadRequest) (PresignedDownload, error) {
	if !validDownloadRequest(request) || s == nil || s.presigner == nil || s.bucket == "" {
		return PresignedDownload{}, ErrInvalidPresignRequest
	}

	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(request.Key),
	}, func(options *s3.PresignOptions) {
		options.Expires = request.ExpiresIn
	})
	if err != nil || result == nil || result.URL == "" {
		return PresignedDownload{}, ErrObjectStorePresignFailed
	}
	return PresignedDownload{URL: result.URL}, nil
}

func validObjectStoreConfig(config configs.UploadConfig) (endpoint, region, bucket, accessKeyID, secretAccessKey string, ok bool) {
	endpoint = strings.TrimSpace(config.ObjectStorageEndpoint)
	region = strings.TrimSpace(config.ObjectStorageRegion)
	bucket = strings.TrimSpace(config.ObjectStorageBucket)
	accessKeyID = strings.TrimSpace(config.ObjectStorageAccessKeyID)
	secretAccessKey = strings.TrimSpace(config.ObjectStorageSecretAccessKey)
	if endpoint == "" || region == "" || bucket == "" || accessKeyID == "" || secretAccessKey == "" {
		return "", "", "", "", "", false
	}

	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", "", "", "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", "", "", "", false
	}
	if (parsed.Scheme == "https") != config.ObjectStorageUseTLS {
		return "", "", "", "", "", false
	}
	return endpoint, region, bucket, accessKeyID, secretAccessKey, true
}

func validUploadRequest(request PresignUploadRequest) bool {
	return strings.TrimSpace(request.Key) != "" &&
		strings.TrimSpace(request.ContentType) != "" &&
		request.ContentLength > 0 &&
		request.ExpiresIn > 0
}

func validDownloadRequest(request PresignDownloadRequest) bool {
	return strings.TrimSpace(request.Key) != "" && request.ExpiresIn > 0
}

func singleValueHeaders(headers map[string][]string) (map[string]string, bool) {
	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) != 1 {
			return nil, false
		}
		result[key] = values[0]
	}
	return result, true
}

var _ ObjectStore = (*S3ObjectStore)(nil)
