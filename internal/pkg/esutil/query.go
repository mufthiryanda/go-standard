package esutil

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MatchQuery builds a full-text match query clause.
func MatchQuery(field, value string) map[string]interface{} {
	return map[string]interface{}{
		"match": map[string]interface{}{
			field: value,
		},
	}
}

// TermQuery builds an exact-match term query clause.
func TermQuery(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
}

// RangeQuery builds a range query clause. Pass nil for from or to omit that bound.
func RangeQuery(field string, from, to interface{}) map[string]interface{} {
	bounds := map[string]interface{}{}
	if from != nil {
		bounds["gte"] = from
	}
	if to != nil {
		bounds["lte"] = to
	}
	return map[string]interface{}{
		"range": map[string]interface{}{
			field: bounds,
		},
	}
}

// BoolQuery builds a compound bool query. Any slice may be nil/empty.
func BoolQuery(
	must, should, filter, mustNot []map[string]interface{},
) map[string]interface{} {
	boolClause := map[string]interface{}{}
	if len(must) > 0 {
		boolClause["must"] = must
	}
	if len(should) > 0 {
		boolClause["should"] = should
	}
	if len(filter) > 0 {
		boolClause["filter"] = filter
	}
	if len(mustNot) > 0 {
		boolClause["must_not"] = mustNot
	}
	return map[string]interface{}{
		"bool": boolClause,
	}
}

// SearchRequest builds a full ES search request body as a *bytes.Reader.
// Returns the body reader and any marshaling error.
func SearchRequest(index string, query map[string]interface{}, from, size int) (*bytes.Reader, error) {
	body := map[string]interface{}{
		"query": query,
		"from":  from,
		"size":  size,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("esutil: marshal search request: %w", err)
	}
	return bytes.NewReader(b), nil
}

// SearchResult is a minimal representation of an ES search response for ID extraction.
type SearchResult struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID string `json:"_id"`
		} `json:"hits"`
	} `json:"hits"`
}

// ExtractIDs pulls _id values from a parsed ES search response.
func ExtractIDs(result SearchResult) []string {
	ids := make([]string, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		ids = append(ids, h.ID)
	}
	return ids
}

// ParseSearchResult unmarshal raw ES response bytes into SearchResult.
func ParseSearchResult(data []byte) (SearchResult, error) {
	var sr SearchResult
	if err := json.Unmarshal(data, &sr); err != nil {
		return sr, fmt.Errorf("esutil: parse search result: %w", err)
	}
	return sr, nil
}
