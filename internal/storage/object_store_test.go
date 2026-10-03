package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFakeObjectStorePresignUploadCapturesRequest(t *testing.T) {
	store := FakeObjectStore{
		UploadResult: PresignedUpload{URL: "https://objects.example.test/uploads/report.pdf", Headers: map[string]string{"Content-Type": "application/pdf"}},
	}
	request := PresignUploadRequest{
		Key:           "uploads/report.pdf",
		ContentType:   "application/pdf",
		ContentLength: 42,
		ExpiresIn:     15 * time.Minute,
	}

	result, err := store.PresignUpload(context.Background(), request)
	if err != nil {
		t.Fatalf("PresignUpload() error = %v; want nil", err)
	}
	if result.URL != store.UploadResult.URL {
		t.Errorf("PresignUpload() URL = %q; want %q", result.URL, store.UploadResult.URL)
	}
	if len(store.UploadRequests) != 1 || store.UploadRequests[0] != request {
		t.Errorf("UploadRequests = %#v; want one captured request %#v", store.UploadRequests, request)
	}
}

func TestFakeObjectStorePresignDownloadCapturesRequestAndPropagatesFailure(t *testing.T) {
	wantErr := errors.New("presigning unavailable")
	store := FakeObjectStore{DownloadErr: wantErr}
	request := PresignDownloadRequest{Key: "uploads/report.pdf", ExpiresIn: 15 * time.Minute}

	_, err := store.PresignDownload(context.Background(), request)
	if !errors.Is(err, wantErr) {
		t.Errorf("PresignDownload() error = %v; want %v", err, wantErr)
	}
	if len(store.DownloadRequests) != 1 || store.DownloadRequests[0] != request {
		t.Errorf("DownloadRequests = %#v; want one captured request %#v", store.DownloadRequests, request)
	}
}

var _ ObjectStore = (*FakeObjectStore)(nil)
