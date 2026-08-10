package jsonapi

import (
	"context"
	"net/url"

	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
)

// GetAllPages fetches every page of a JSON:API collection by following links.next.
func GetAllPages[T any](ctx context.Context, c *httpclient.Client, initialPath string) ([]httpclient.Resource[T], error) {
	var all []httpclient.Resource[T]
	path := initialPath
	for path != "" {
		col, err := httpclient.GetJSONAPICollection[T](ctx, c, path)
		if err != nil {
			return nil, err
		}
		all = append(all, col.Data...)
		next := col.Links.Next
		if next == "" {
			break
		}
		path = extractPath(next)
		if path == "" {
			break
		}
	}
	return all, nil
}

// extractPath returns the request-URI (path + query) from a potentially absolute URL.
// If s is already a path, it is returned unchanged.
func extractPath(s string) string {
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || !u.IsAbs() {
		return s
	}
	return u.RequestURI()
}
