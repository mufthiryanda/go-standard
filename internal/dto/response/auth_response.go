package response

import "time"

// AuthTokenResponse is the API-facing shape for authentication token pairs.
type AuthTokenResponse struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt    string `json:"expires_at"    example:"2025-07-01T12:15:00+07:00"`
}

// NewAuthTokenResponse constructs an AuthTokenResponse, formatting expiresAt as ISO8601.
func NewAuthTokenResponse(accessToken, refreshToken string, expiresAt time.Time) AuthTokenResponse {
	return AuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.In(jakartaLoc).Format(iso8601),
	}
}
