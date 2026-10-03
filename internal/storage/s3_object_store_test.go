package storage

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nick130920/fintech-backend/configs"
)

type fakeS3Presigner struct {
	putResult  *v4.PresignedHTTPRequest
	putErr     error
	getResult  *v4.PresignedHTTPRequest
	getErr     error
	putInput   *s3.PutObjectInput
	putExpires time.Duration
	getInput   *s3.GetObjectInput
	getExpires time.Duration
}

func (p *fakeS3Presigner) PresignPutObject(_ context.Context, input *s3.PutObjectInput, options ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	p.putInput = input
	presignOptions := s3.PresignOptions{}
	for _, option := range options {
		option(&presignOptions)
	}
	p.putExpires = presignOptions.Expires
	return p.putResult, p.putErr
}

func (p *fakeS3Presigner) PresignGetObject(_ context.Context, input *s3.GetObjectInput, options ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	p.getInput = input
	presignOptions := s3.PresignOptions{}
	for _, option := range options {
		option(&presignOptions)
	}
	p.getExpires = presignOptions.Expires
	return p.getResult, p.getErr
}

func TestS3ObjectStorePresignUploadForwardsRequestAndReturnsSignedHeaders(t *testing.T) {
	presigner := &fakeS3Presigner{
		putResult: &v4.PresignedHTTPRequest{
			URL: "https://objects.example.test/reports/annual.pdf?X-Amz-Signature=redacted",
			SignedHeader: http.Header{
				"Content-Type":     {"application/pdf"},
				"X-Amz-Meta-Trace": {"trace-123"},
			},
		},
	}
	store := newS3ObjectStoreWithPresigner("documents", presigner)
	request := PresignUploadRequest{
		Key:           "reports/annual.pdf",
		ContentType:   "application/pdf",
		ContentLength: 42,
		ExpiresIn:     15 * time.Minute,
	}

	result, err := store.PresignUpload(context.Background(), request)
	if err != nil {
		t.Fatalf("PresignUpload() error = %v; want nil", err)
	}
	if result.URL != presigner.putResult.URL {
		t.Errorf("PresignUpload() URL = %q; want %q", result.URL, presigner.putResult.URL)
	}
	if got, want := result.Headers, map[string]string{"Content-Type": "application/pdf", "X-Amz-Meta-Trace": "trace-123"}; !equalHeaders(got, want) {
		t.Errorf("PresignUpload() Headers = %#v; want %#v", got, want)
	}
	if got := aws.ToString(presigner.putInput.Bucket); got != "documents" {
		t.Errorf("put bucket = %q; want documents", got)
	}
	if got := aws.ToString(presigner.putInput.Key); got != request.Key {
		t.Errorf("put key = %q; want %q", got, request.Key)
	}
	if got := aws.ToString(presigner.putInput.ContentType); got != request.ContentType {
		t.Errorf("put content type = %q; want %q", got, request.ContentType)
	}
	if got := aws.ToInt64(presigner.putInput.ContentLength); got != request.ContentLength {
		t.Errorf("put content length = %d; want %d", got, request.ContentLength)
	}
	if presigner.putExpires != request.ExpiresIn {
		t.Errorf("put expiry = %s; want %s", presigner.putExpires, request.ExpiresIn)
	}
}

func TestS3ObjectStorePresignDownloadForwardsRequest(t *testing.T) {
	presigner := &fakeS3Presigner{
		getResult: &v4.PresignedHTTPRequest{URL: "https://objects.example.test/reports/annual.pdf?X-Amz-Signature=redacted"},
	}
	store := newS3ObjectStoreWithPresigner("documents", presigner)
	request := PresignDownloadRequest{Key: "reports/annual.pdf", ExpiresIn: 10 * time.Minute}

	result, err := store.PresignDownload(context.Background(), request)
	if err != nil {
		t.Fatalf("PresignDownload() error = %v; want nil", err)
	}
	if result.URL != presigner.getResult.URL {
		t.Errorf("PresignDownload() URL = %q; want %q", result.URL, presigner.getResult.URL)
	}
	if got := aws.ToString(presigner.getInput.Bucket); got != "documents" {
		t.Errorf("get bucket = %q; want documents", got)
	}
	if got := aws.ToString(presigner.getInput.Key); got != request.Key {
		t.Errorf("get key = %q; want %q", got, request.Key)
	}
	if presigner.getExpires != request.ExpiresIn {
		t.Errorf("get expiry = %s; want %s", presigner.getExpires, request.ExpiresIn)
	}
}

func TestS3ObjectStoreRejectsInvalidRequestsWithoutCallingPresigner(t *testing.T) {
	tests := []struct {
		name     string
		upload   *PresignUploadRequest
		download *PresignDownloadRequest
	}{
		{name: "upload empty key", upload: &PresignUploadRequest{ContentType: "image/png", ContentLength: 1, ExpiresIn: time.Minute}},
		{name: "upload empty content type", upload: &PresignUploadRequest{Key: "images/a.png", ContentLength: 1, ExpiresIn: time.Minute}},
		{name: "upload nonpositive content length", upload: &PresignUploadRequest{Key: "images/a.png", ContentType: "image/png", ExpiresIn: time.Minute}},
		{name: "upload nonpositive expiry", upload: &PresignUploadRequest{Key: "images/a.png", ContentType: "image/png", ContentLength: 1}},
		{name: "download empty key", download: &PresignDownloadRequest{ExpiresIn: time.Minute}},
		{name: "download nonpositive expiry", download: &PresignDownloadRequest{Key: "images/a.png"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			presigner := &fakeS3Presigner{}
			store := newS3ObjectStoreWithPresigner("documents", presigner)

			var err error
			if test.upload != nil {
				_, err = store.PresignUpload(context.Background(), *test.upload)
			} else {
				_, err = store.PresignDownload(context.Background(), *test.download)
			}
			if !errors.Is(err, ErrInvalidPresignRequest) {
				t.Errorf("error = %v; want ErrInvalidPresignRequest", err)
			}
			if presigner.putInput != nil || presigner.getInput != nil {
				t.Errorf("presigner was called: put=%#v get=%#v", presigner.putInput, presigner.getInput)
			}
		})
	}
}

func TestS3ObjectStoreSanitizesProviderAndUnsupportedHeaderErrors(t *testing.T) {
	tests := []struct {
		name      string
		presigner *fakeS3Presigner
	}{
		{
			name:      "upload provider error",
			presigner: &fakeS3Presigner{putErr: errors.New("https://secret.example.test?X-Amz-Signature=secret")},
		},
		{
			name:      "download provider error",
			presigner: &fakeS3Presigner{getErr: errors.New("credential=secret")},
		},
		{
			name:      "multiple signed header values",
			presigner: &fakeS3Presigner{putResult: &v4.PresignedHTTPRequest{URL: "https://objects.example.test", SignedHeader: http.Header{"X-Amz-Meta-Tag": {"one", "two"}}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newS3ObjectStoreWithPresigner("documents", test.presigner)
			var err error
			if strings.Contains(test.name, "download") {
				_, err = store.PresignDownload(context.Background(), PresignDownloadRequest{Key: "reports/annual.pdf", ExpiresIn: time.Minute})
			} else {
				_, err = store.PresignUpload(context.Background(), PresignUploadRequest{Key: "reports/annual.pdf", ContentType: "application/pdf", ContentLength: 42, ExpiresIn: time.Minute})
			}
			if !errors.Is(err, ErrObjectStorePresignFailed) {
				t.Errorf("error = %v; want ErrObjectStorePresignFailed", err)
			}
			if got := err.Error(); strings.Contains(got, "secret") || strings.Contains(got, "objects.example.test") || strings.Contains(got, "credential") {
				t.Errorf("error exposes provider details: %q", got)
			}
		})
	}
}

func TestNewS3ObjectStoreRejectsIncompleteOrIncoherentConfiguration(t *testing.T) {
	valid := configs.UploadConfig{
		ObjectStorageEndpoint:        "https://account.r2.cloudflarestorage.com",
		ObjectStorageRegion:          "auto",
		ObjectStorageBucket:          "documents",
		ObjectStorageAccessKeyID:     "access-key",
		ObjectStorageSecretAccessKey: "secret-key",
		ObjectStorageUseTLS:          true,
		ObjectStorageForcePathStyle:  true,
	}
	tests := []struct {
		name string
		edit func(*configs.UploadConfig)
	}{
		{name: "missing endpoint", edit: func(config *configs.UploadConfig) { config.ObjectStorageEndpoint = "" }},
		{name: "invalid endpoint", edit: func(config *configs.UploadConfig) { config.ObjectStorageEndpoint = "not-a-url" }},
		{name: "TLS mismatch", edit: func(config *configs.UploadConfig) {
			config.ObjectStorageEndpoint = "http://account.r2.cloudflarestorage.com"
		}},
		{name: "missing region", edit: func(config *configs.UploadConfig) { config.ObjectStorageRegion = "" }},
		{name: "missing bucket", edit: func(config *configs.UploadConfig) { config.ObjectStorageBucket = "" }},
		{name: "missing access key", edit: func(config *configs.UploadConfig) { config.ObjectStorageAccessKeyID = "" }},
		{name: "missing secret key", edit: func(config *configs.UploadConfig) { config.ObjectStorageSecretAccessKey = "" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.edit(&config)
			store, err := NewS3ObjectStore(config)
			if !errors.Is(err, ErrInvalidObjectStoreConfig) {
				t.Errorf("NewS3ObjectStore() error = %v; want ErrInvalidObjectStoreConfig", err)
			}
			if store != nil {
				t.Errorf("NewS3ObjectStore() store = %#v; want nil", store)
			}
		})
	}
}

func TestNewS3ObjectStorePresignsLocallyWithConfiguredS3Addressing(t *testing.T) {
	const (
		endpoint    = "https://account.r2.cloudflarestorage.com"
		bucket      = "documents"
		key         = "reports/annual.pdf"
		accessKeyID = "dummy-access-key"
		region      = "auto"
	)
	request := PresignUploadRequest{
		Key:           key,
		ContentType:   "application/pdf",
		ContentLength: 42,
		ExpiresIn:     15 * time.Minute,
	}
	tests := []struct {
		name           string
		forcePathStyle bool
		wantHost       string
		wantPath       string
	}{
		{
			name:           "path style",
			forcePathStyle: true,
			wantHost:       "account.r2.cloudflarestorage.com",
			wantPath:       "/documents/reports/annual.pdf",
		},
		{
			name:           "virtual host style",
			forcePathStyle: false,
			wantHost:       "documents.account.r2.cloudflarestorage.com",
			wantPath:       "/reports/annual.pdf",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := NewS3ObjectStore(configs.UploadConfig{
				ObjectStorageEndpoint:        endpoint,
				ObjectStorageRegion:          region,
				ObjectStorageBucket:          bucket,
				ObjectStorageAccessKeyID:     accessKeyID,
				ObjectStorageSecretAccessKey: "dummy-secret-key",
				ObjectStorageUseTLS:          true,
				ObjectStorageForcePathStyle:  test.forcePathStyle,
			})
			if err != nil {
				t.Fatalf("NewS3ObjectStore() error = %v; want nil", err)
			}

			result, err := store.PresignUpload(context.Background(), request)
			if err != nil {
				t.Fatalf("PresignUpload() error = %v; want nil", err)
			}
			presignedURL, err := url.Parse(result.URL)
			if err != nil {
				t.Fatalf("url.Parse(PresignUpload().URL) error = %v", err)
			}
			if got := presignedURL.Scheme; got != "https" {
				t.Errorf("presigned URL scheme = %q; want https", got)
			}
			if got := presignedURL.Host; got != test.wantHost {
				t.Errorf("presigned URL host = %q; want %q", got, test.wantHost)
			}
			if got := presignedURL.Path; got != test.wantPath {
				t.Errorf("presigned URL path = %q; want %q", got, test.wantPath)
			}

			query := presignedURL.Query()
			if got := query.Get("X-Amz-Expires"); got != "900" {
				t.Errorf("X-Amz-Expires = %q; want 900", got)
			}
			credential := query.Get("X-Amz-Credential")
			if !strings.HasPrefix(credential, accessKeyID+"/") || !strings.HasSuffix(credential, "/"+region+"/s3/aws4_request") {
				t.Errorf("X-Amz-Credential scope = %q; want %q prefix and %q suffix", credential, accessKeyID+"/", "/"+region+"/s3/aws4_request")
			}
			signedHeaders := query.Get("X-Amz-SignedHeaders")
			if !strings.Contains(signedHeaders, "content-length") || !strings.Contains(signedHeaders, "content-type") {
				t.Errorf("X-Amz-SignedHeaders = %q; want content-length and content-type", signedHeaders)
			}
			if got := result.Headers["Content-Length"]; got != "42" {
				t.Errorf("signed Content-Length = %q; want 42", got)
			}
			if got := result.Headers["Content-Type"]; got != request.ContentType {
				t.Errorf("signed Content-Type = %q; want %q", got, request.ContentType)
			}
		})
	}
}

func TestNewS3ObjectStoreSatisfiesObjectStore(t *testing.T) {
	var _ ObjectStore = (*S3ObjectStore)(nil)
}

func equalHeaders(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			return false
		}
	}
	return true
}
