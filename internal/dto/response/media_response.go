package response

// MediaUploadResponse is returned after a successful file upload.
type MediaUploadResponse struct {
	Key       string `json:"key"`
	PublicURL string `json:"public_url"`
}

// PresignedURLResponse is returned when a presigned PUT URL is generated.
type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	Key       string `json:"key"`
	ExpiresIn int    `json:"expires_in"` // seconds
}
