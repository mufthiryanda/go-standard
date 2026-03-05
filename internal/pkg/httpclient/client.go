package httpclient

import (
	"bytes"
	"context"
	"net/http"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"go.uber.org/zap"
)

// Client is the outbound HTTP client interface.
// All integrations depend on this interface, never the concrete struct.
type Client interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error)
	Post(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error)
	Put(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error)
	Patch(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error)
	Delete(ctx context.Context, url string, headers map[string]string) (*http.Response, error)
}

type baseClient struct {
	httpClient *http.Client
	logger     *zap.Logger
	retry      RetryConfig
	cb         *circuitBreaker
	timeout    time.Duration
}

// NewBaseClient creates a new Client using cleanhttp's pooled transport.
func NewBaseClient(logger *zap.Logger, opts ...Option) Client {
	c := &baseClient{
		httpClient: cleanhttp.DefaultPooledClient(),
		logger:     logger,
		timeout:    30 * time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *baseClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	start := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req = req.WithContext(reqCtx)

	c.logRequest(req)
	res, err := c.executeWithResilience(req)
	c.logResponse(req, res, time.Since(start), err)

	if err != nil {
		return nil, MapError(err)
	}
	return res, nil
}

func (c *baseClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, MapError(err)
	}
	applyHeaders(req, headers)
	return c.Do(ctx, req)
}

func (c *baseClient) Post(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, MapError(err)
	}
	applyHeaders(req, headers)
	return c.Do(ctx, req)
}

func (c *baseClient) Put(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, MapError(err)
	}
	applyHeaders(req, headers)
	return c.Do(ctx, req)
}

func (c *baseClient) Patch(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, MapError(err)
	}
	applyHeaders(req, headers)
	return c.Do(ctx, req)
}

func (c *baseClient) Delete(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, MapError(err)
	}
	applyHeaders(req, headers)
	return c.Do(ctx, req)
}

func applyHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}
