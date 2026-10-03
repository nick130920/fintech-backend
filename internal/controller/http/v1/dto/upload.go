package dto

// PresignUploadRequest contains the client-controlled properties of an upload.
type PresignUploadRequest struct {
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
}

// PresignUploadResponse contains a presigned PUT authorization.
type PresignUploadResponse struct {
	ObjectRef        string            `json:"object_ref"`
	UploadURL        string            `json:"upload_url"`
	Headers          map[string]string `json:"headers"`
	ExpiresInSeconds int               `json:"expires_in_seconds"`
}
