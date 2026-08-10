package jsonapi

import "net/http"

// Transport is an http.RoundTripper that enforces JSON:API media types.
// It overrides Accept on all requests and Content-Type on PATCH requests.
// ctlkit's Patch() sends Content-Type: application/json; this middleware corrects it.
type Transport struct {
	Wrapped http.RoundTripper
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.Header.Set("Accept", "application/vnd.api+json")
	if r2.Method == http.MethodPatch {
		r2.Header.Set("Content-Type", "application/vnd.api+json")
	}
	wrapped := t.Wrapped
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	return wrapped.RoundTrip(r2)
}
