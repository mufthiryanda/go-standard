package jwt

import (
	"errors"
	"fmt"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/config"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims extends RegisteredClaims with the user role.
type Claims struct {
	gojwt.RegisteredClaims
	Role string `json:"role,omitempty"`
}

// Manager handles JWT generation and validation.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
	jakartaLoc *time.Location
}

// NewManager constructs a Manager, parsing TTL strings from config.
func NewManager(cfg *config.Config) (*Manager, error) {
	accessTTL, err := time.ParseDuration(cfg.JWT.AccessTTL)
	if err != nil {
		return nil, fmt.Errorf("jwt: invalid access_ttl %q: %w", cfg.JWT.AccessTTL, err)
	}

	refreshTTL, err := time.ParseDuration(cfg.JWT.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("jwt: invalid refresh_ttl %q: %w", cfg.JWT.RefreshTTL, err)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return nil, fmt.Errorf("jwt: failed to load timezone: %w", err)
	}

	return &Manager{
		secret:     []byte(cfg.JWT.Secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     cfg.JWT.Issuer,
		jakartaLoc: loc,
	}, nil
}

// GenerateAccessToken issues a signed HS256 access token for the given user.
func (m *Manager) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now().In(m.jakartaLoc)
	claims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.accessTTL)),
		},
		Role: role,
	}
	return gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// GenerateRefreshToken issues a signed HS256 refresh token for the given user.
func (m *Manager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now().In(m.jakartaLoc)
	claims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	}
	return gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ValidateToken parses and validates a token string, returning its claims.
func (m *Manager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenStr, &Claims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		if errors.Is(err, gojwt.ErrTokenExpired) {
			return nil, apperror.Unauthorized("token expired")
		}
		return nil, apperror.Unauthorized("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, apperror.Unauthorized("invalid token claims")
	}
	return claims, nil
}
