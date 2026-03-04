package infrastructure

import (
	"fmt"
	"net/http"

	"go-standard/internal/apperror"
	"go-standard/internal/config"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

// NewElasticClient creates a go-elasticsearch/v8 client, verifies
// connectivity via a cluster info call, and returns a no-op cleanup function.
// The ES client holds no persistent connection that requires explicit closing.
// Satisfies the Wire (T, func(), error) cleanup pattern.
func NewElasticClient(cfg *config.Config) (*elasticsearch.Client, func(), error) {
	esCfg := buildElasticConfig(cfg.Elasticsearch)

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("elasticsearch: failed to create client", err)
	}

	res, err := client.Info()
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("elasticsearch: info request failed", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, nil, apperror.ServiceUnavailable(
			fmt.Sprintf("elasticsearch: info returned non-2xx status: %s", res.Status()),
			nil,
		)
	}

	if res.StatusCode != http.StatusOK {
		return nil, nil, apperror.ServiceUnavailable(
			fmt.Sprintf("elasticsearch: unexpected status code %d", res.StatusCode),
			nil,
		)
	}

	zap.L().Info("elasticsearch: connection established",
		zap.Strings("addresses", cfg.Elasticsearch.Addresses),
	)

	// The ES HTTP client manages its own connection pool internally.
	// No explicit close is required or supported.
	cleanup := func() {
		zap.L().Info("elasticsearch: client released (no-op close)")
	}

	return client, cleanup, nil
}

// buildElasticConfig maps config fields to elasticsearch.Config.
func buildElasticConfig(cfg config.Elasticsearch) elasticsearch.Config {
	return elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}
}
