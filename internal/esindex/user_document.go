package esindex

import (
	"bytes"
	"encoding/json"
	"io"

	"go-standard/internal/domain/model"
)

// UserDocument is the Elasticsearch representation of a user.
// It is separate from the domain model to decouple ES schema from DB schema.
type UserDocument struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	Phone     string  `json:"phone,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DeletedAt *string `json:"deleted_at,omitempty"`
}

// UserDocumentFromModel converts a domain User model into a UserDocument.
func UserDocumentFromModel(u *model.User) UserDocument {
	doc := UserDocument{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Safely handle the Phone pointer
	if u.Phone != nil {
		doc.Phone = *u.Phone
	}

	// Safely handle the DeletedAt pointer
	if u.DeletedAt != nil {
		s := u.DeletedAt.Format("2006-01-02T15:04:05Z07:00")
		doc.DeletedAt = &s
	}

	return doc
}

// ToReader marshals the document to JSON and returns it as an io.Reader
// suitable for use with the go-elasticsearch client.
func (d UserDocument) ToReader() (io.Reader, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}
