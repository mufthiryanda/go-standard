package response

import "time"

// AuthTokenResponse is the API-facing shape for authentication token pairs.
type AuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

// NewAuthTokenResponse constructs an AuthTokenResponse, formatting expiresAt as ISO8601.
func NewAuthTokenResponse(accessToken, refreshToken string, expiresAt time.Time) AuthTokenResponse {
	return AuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.In(jakartaLoc).Format(iso8601),
	}
}
