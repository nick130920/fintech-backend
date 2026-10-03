package storage

import (
	"context"
	"time"
)

// ObjectStore creates presigned object-storage URLs for uploads and downloads.
type ObjectStore interface {
	PresignUpload(context.Context, PresignUploadRequest) (PresignedUpload, error)
	PresignDownload(context.Context, PresignDownloadRequest) (PresignedDownload, error)
}

// PresignUploadRequest describes an object upload authorization.
type PresignUploadRequest struct {
	Key           string
	ContentType   string
	ContentLength int64
	ExpiresIn     time.Duration
}

// PresignedUpload is the authorization needed to upload an object.
type PresignedUpload struct {
	URL     string
	Headers map[string]string
}

// PresignDownloadRequest describes an object download authorization.
type PresignDownloadRequest struct {
	Key       string
	ExpiresIn time.Duration
}

// PresignedDownload is the authorization needed to download an object.
type PresignedDownload struct {
	URL string
}

// FakeObjectStore is a deterministic ObjectStore test double.
type FakeObjectStore struct {
	UploadResult     PresignedUpload
	UploadErr        error
	DownloadResult   PresignedDownload
	DownloadErr      error
	UploadRequests   []PresignUploadRequest
	DownloadRequests []PresignDownloadRequest
}

func (s *FakeObjectStore) PresignUpload(_ context.Context, request PresignUploadRequest) (PresignedUpload, error) {
	s.UploadRequests = append(s.UploadRequests, request)
	if s.UploadErr != nil {
		return PresignedUpload{}, s.UploadErr
	}
	return s.UploadResult, nil
}

func (s *FakeObjectStore) PresignDownload(_ context.Context, request PresignDownloadRequest) (PresignedDownload, error) {
	s.DownloadRequests = append(s.DownloadRequests, request)
	if s.DownloadErr != nil {
		return PresignedDownload{}, s.DownloadErr
	}
	return s.DownloadResult, nil
}
