// Package jsonapi provides JSON:API envelope helpers for the ClusterControl CLI.
package jsonapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
)

// Request wraps attributes in a JSON:API request body envelope.
type Request[T any] struct {
	Data RequestData[T] `json:"data"`
}

// RequestData is the "data" key of a JSON:API request body.
type RequestData[T any] struct {
	Type       string `json:"type"`
	Attributes T      `json:"attributes"`
}

// Wrap constructs a JSON:API request body for create/update operations.
func Wrap[T any](resourceType string, attrs T) Request[T] {
	return Request[T]{Data: RequestData[T]{Type: resourceType, Attributes: attrs}}
}

// PatchSingle sends a PATCH request with body and decodes the response as a JSON:API single resource.
// The jsonapi.Transport middleware (injected via the HTTP client) upgrades Content-Type to
// application/vnd.api+json on the wire; this function handles response decoding.
func PatchSingle[T any](ctx context.Context, c *httpclient.Client, path string, body any) (httpclient.Resource[T], error) {
	var doc struct {
		Data struct {
			ID         string          `json:"id"`
			Type       string          `json:"type"`
			Attributes json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := c.Patch(ctx, path, body, &doc); err != nil {
		return httpclient.Resource[T]{}, err
	}
	var attrs T
	if len(doc.Data.Attributes) > 0 {
		if err := json.Unmarshal(doc.Data.Attributes, &attrs); err != nil {
			return httpclient.Resource[T]{}, fmt.Errorf("decoding attributes: %w", err)
		}
	}
	return httpclient.Resource[T]{ID: doc.Data.ID, Type: doc.Data.Type, Attributes: attrs}, nil
}
