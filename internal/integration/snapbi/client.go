package snapbi

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/config"
	"go-standard/internal/pkg/cache"
	"go-standard/internal/pkg/httpclient"
	"go-standard/internal/pkg/rediskey"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SnapBIClient is the interface for SNAP BI integration operations.
type SnapBIClient interface {
	GetAccessToken(ctx context.Context) (string, error)
	TransferVA(ctx context.Context, req TransferVARequest) (*TransferVAResponse, error)
	InquiryVA(ctx context.Context, req InquiryVARequest) (*InquiryVAResponse, error)
}

type snapBIClient struct {
	base       httpclient.Client
	cfg        config.SnapBI
	privateKey *rsa.PrivateKey
	logger     *zap.Logger
	rdb        *redis.Client
	jakartaLoc *time.Location
}

// NewSnapBIClient creates a SnapBIClient. Returns a cleanup func and error per Wire convention.
func NewSnapBIClient(
	base httpclient.Client,
	cfg config.SnapBI,
	rdb *redis.Client,
	logger *zap.Logger,
) (SnapBIClient, func(), error) {
	pk, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, nil, apperror.Internal("snapbi: failed to load private key", err)
	}
	loc := mustLoadJakarta()
	cleanup := func() {
		logger.Info("snapbi: client shutdown")
	}
	return &snapBIClient{
		base:       base,
		cfg:        cfg,
		privateKey: pk,
		logger:     logger,
		rdb:        rdb,
		jakartaLoc: loc,
	}, cleanup, nil
}

// GetAccessToken retrieves the B2B access token, using Redis cache when possible.
func (c *snapBIClient) GetAccessToken(ctx context.Context) (string, error) {
	ttl := time.Duration(c.cfg.AccessTokenTTL-60) * time.Second
	if ttl <= 0 {
		ttl = 780 * time.Second // safe fallback
	}
	token, err := cache.GetOrLoad[string](ctx, c.rdb, rediskey.SnapBIAccessToken(), ttl, func() (string, error) {
		return c.fetchAccessToken(ctx)
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (c *snapBIClient) fetchAccessToken(ctx context.Context) (string, error) {
	req, err := c.buildAccessTokenRequest(ctx)
	if err != nil {
		return "", err
	}
	resp, err := c.base.Do(ctx, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apperror.Internal("snapbi: read access token response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", ParseSnapBIError(body)
	}
	var result AccessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", apperror.Internal("snapbi: decode access token response", err)
	}
	return result.AccessToken, nil
}

func (c *snapBIClient) buildAccessTokenRequest(ctx context.Context) (*http.Request, error) {
	timestamp := BuildTimestamp(c.jakartaLoc)
	sig, err := SignAsymmetric(c.privateKey, c.cfg.ClientKey, timestamp)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(AccessTokenRequest{GrantType: "client_credentials"})
	if err != nil {
		return nil, apperror.Internal("snapbi: marshal access token request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/snap/v1.0/access-token/b2b", bytes.NewReader(payload))
	if err != nil {
		return nil, apperror.Internal("snapbi: build access token request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CLIENT-KEY", c.cfg.ClientKey)
	req.Header.Set("X-TIMESTAMP", timestamp)
	req.Header.Set("X-SIGNATURE", sig)
	return req, nil
}

// TransferVA executes a VA payment transfer via SNAP BI.
func (c *snapBIClient) TransferVA(ctx context.Context, reqDTO TransferVARequest) (*TransferVAResponse, error) {
	body, err := json.Marshal(reqDTO)
	if err != nil {
		return nil, apperror.Internal("snapbi: marshal transfer va request", err)
	}
	req, err := c.buildTransactionRequest(ctx, http.MethodPost, "/snap/v1.0/transfer-va/payment", body)
	if err != nil {
		return nil, err
	}
	resp, err := c.base.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.Internal("snapbi: read transfer va response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseSnapBIError(respBody)
	}
	var result TransferVAResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, apperror.Internal("snapbi: decode transfer va response", err)
	}
	return &result, nil
}

// InquiryVA inquires about a VA via SNAP BI.
func (c *snapBIClient) InquiryVA(ctx context.Context, reqDTO InquiryVARequest) (*InquiryVAResponse, error) {
	body, err := json.Marshal(reqDTO)
	if err != nil {
		return nil, apperror.Internal("snapbi: marshal inquiry va request", err)
	}
	req, err := c.buildTransactionRequest(ctx, http.MethodPost, "/snap/v1.0/transfer-va/inquiry", body)
	if err != nil {
		return nil, err
	}
	resp, err := c.base.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.Internal("snapbi: read inquiry va response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseSnapBIError(respBody)
	}
	var result InquiryVAResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, apperror.Internal("snapbi: decode inquiry va response", err)
	}
	return &result, nil
}

func (c *snapBIClient) buildTransactionRequest(
	ctx context.Context,
	method, path string,
	body []byte,
) (*http.Request, error) {
	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	timestamp := BuildTimestamp(c.jakartaLoc)
	externalID := BuildExternalID()

	sig, err := SignSymmetric(accessToken, method, path, body, timestamp)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, apperror.Internal("snapbi: build transaction request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-TIMESTAMP", timestamp)
	req.Header.Set("X-CLIENT-KEY", c.cfg.ClientKey)
	req.Header.Set("X-PARTNER-ID", c.cfg.PartnerID)
	req.Header.Set("X-EXTERNAL-ID", externalID)
	req.Header.Set("CHANNEL-ID", c.cfg.ChannelID)
	req.Header.Set("X-SIGNATURE", sig)
	return req, nil
}
