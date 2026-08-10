package jsonapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
)

// ResourceWithSelf is a JSON:API resource that includes the server-provided links.self URL.
type ResourceWithSelf[T any] struct {
	httpclient.Resource[T]
	SelfLink string
}

type selfRawResource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes json.RawMessage `json:"attributes"`
	Links      struct {
		Self string `json:"self"`
	} `json:"links"`
}

type selfSingleDoc struct {
	Data selfRawResource `json:"data"`
}

// GetSingle fetches a single JSON:API resource and captures data.links.self.
// Uses c.Get which goes through jsonapi.Transport, so Accept is set to
// application/vnd.api+json on the wire.
func GetSingle[T any](ctx context.Context, c *httpclient.Client, path string) (ResourceWithSelf[T], error) {
	var doc selfSingleDoc
	if err := c.Get(ctx, path, &doc); err != nil {
		return ResourceWithSelf[T]{}, err
	}
	var attrs T
	if len(doc.Data.Attributes) > 0 {
		if err := json.Unmarshal(doc.Data.Attributes, &attrs); err != nil {
			return ResourceWithSelf[T]{}, fmt.Errorf("decoding attributes: %w", err)
		}
	}
	return ResourceWithSelf[T]{
		Resource: httpclient.Resource[T]{ID: doc.Data.ID, Type: doc.Data.Type, Attributes: attrs},
		SelfLink: doc.Data.Links.Self,
	}, nil
}

// SelfPath returns the request-URI (path + query) from links.self.
// Falls back to fallback if selfLink is empty.
// If selfLink is an absolute URL, only the path+query is returned.
func SelfPath(selfLink, fallback string) string {
	if selfLink == "" {
		return fallback
	}
	return extractPath(selfLink)
}
