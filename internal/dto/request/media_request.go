package request

// GenerateUploadURLRequest holds the parameters for requesting a presigned PUT URL.
type GenerateUploadURLRequest struct {
	Entity      string `json:"entity"       validate:"required"`
	Filename    string `json:"filename"     validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}
