// internal/esindex/bootstrap.go
package esindex

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

//go:embed user_mapping.json
var userMappingJSON []byte

type indexConfig struct {
	alias      string
	index      string
	mappingRaw []byte
}

var indexes = []indexConfig{
	{
		alias:      "project_users",
		index:      "project_users_v1",
		mappingRaw: userMappingJSON,
	},
}

// Bootstrap ensures all required ES indexes and aliases exist.
// Safe to call on every app start — idempotent.
func Bootstrap(es *elasticsearch.Client) error {
	ctx := context.Background()
	for _, cfg := range indexes {
		if err := ensureIndex(ctx, es, cfg); err != nil {
			return fmt.Errorf("esindex bootstrap: %s: %w", cfg.alias, err)
		}
	}
	return nil
}

func ensureIndex(ctx context.Context, es *elasticsearch.Client, cfg indexConfig) error {
	aliasExists, err := checkAliasExists(ctx, es, cfg.alias)
	if err != nil {
		return err
	}

	if aliasExists {
		zap.L().Info("esindex: alias already exists, skipping", zap.String("alias", cfg.alias))
		return nil
	}

	if err := createIndex(ctx, es, cfg.index, cfg.mappingRaw); err != nil {
		return err
	}

	if err := createAlias(ctx, es, cfg.index, cfg.alias); err != nil {
		return err
	}

	zap.L().Info("esindex: index and alias created",
		zap.String("index", cfg.index),
		zap.String("alias", cfg.alias),
	)
	return nil
}

func checkAliasExists(ctx context.Context, es *elasticsearch.Client, alias string) (bool, error) {
	res, err := es.Indices.ExistsAlias([]string{alias}, es.Indices.ExistsAlias.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("check alias %q: %w", alias, err)
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK, nil
}

func createIndex(ctx context.Context, es *elasticsearch.Client, index string, mapping []byte) error {
	res, err := es.Indices.Create(
		index,
		es.Indices.Create.WithContext(ctx),
		es.Indices.Create.WithBody(bytes.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("create index %q: %w", index, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("create index %q returned %s", index, res.Status())
	}
	return nil
}

func createAlias(ctx context.Context, es *elasticsearch.Client, index, alias string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"actions": []map[string]interface{}{
			{"add": map[string]interface{}{"index": index, "alias": alias}},
		},
	})

	res, err := es.Indices.UpdateAliases(
		bytes.NewReader(body),
		es.Indices.UpdateAliases.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("create alias %q: %w", alias, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("create alias %q returned %s", alias, res.Status())
	}
	return nil
}
