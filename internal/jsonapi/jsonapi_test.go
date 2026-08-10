package jsonapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotdevlabs/ctlkit/pkg/httpclient"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

// captureTransport records the last request it received and delegates to Wrapped.
type captureTransport struct {
	Wrapped http.RoundTripper
	last    *http.Request
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.last = r
	wrapped := c.Wrapped
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	return wrapped.RoundTrip(r)
}

// TestWrap_Serialization asserts Wrap produces a valid JSON:API request envelope.
func TestWrap_Serialization(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}
	wrapped := jsonapi.Wrap("packages", attrs{Name: "pkg"})
	raw, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 {
		t.Errorf("expected exactly one top-level key 'data', got keys: %v", body)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be object, got %T", body["data"])
	}
	if data["type"] != "packages" {
		t.Errorf("expected data.type=packages, got %v", data["type"])
	}
	attrs2, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected data.attributes to be object, got %T", data["attributes"])
	}
	if attrs2["name"] != "pkg" {
		t.Errorf("expected attributes.name=pkg, got %v", attrs2["name"])
	}
}

// TestWrap_TypeField verifies the type string is preserved as-is.
func TestWrap_TypeField(t *testing.T) {
	type attrs struct{}
	wrapped := jsonapi.Wrap("project_secrets", attrs{})
	if wrapped.Data.Type != "project_secrets" {
		t.Errorf("expected type=project_secrets, got %q", wrapped.Data.Type)
	}
}

// TestTransport_SetsAcceptAlways verifies Accept is set on all requests.
func TestTransport_SetsAcceptAlways(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cap := &captureTransport{Wrapped: http.DefaultTransport}
	tr := &jsonapi.Transport{Wrapped: cap}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := cap.last.Header.Get("Accept"); got != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", got)
	}
}

// TestTransport_SetsContentTypeForPATCH verifies PATCH gets Content-Type.
func TestTransport_SetsContentTypeForPATCH(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cap := &captureTransport{Wrapped: http.DefaultTransport}
	tr := &jsonapi.Transport{Wrapped: cap}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPatch, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := cap.last.Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got)
	}
}

// TestTransport_SetsContentTypeForPOST verifies POST also gets Content-Type.
func TestTransport_SetsContentTypeForPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cap := &captureTransport{Wrapped: http.DefaultTransport}
	tr := &jsonapi.Transport{Wrapped: cap}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := cap.last.Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got)
	}
}

// TestTransport_DoesNotSetContentTypeForGET verifies GET has no Content-Type body header.
func TestTransport_DoesNotSetContentTypeForGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cap := &captureTransport{Wrapped: http.DefaultTransport}
	tr := &jsonapi.Transport{Wrapped: cap}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := cap.last.Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want empty for GET", got)
	}
}

// TestPatchSingle_DecodesResponse verifies PatchSingle decodes a JSON:API response.
func TestPatchSingle_DecodesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"1","type":"packages","attributes":{"name":"mypkg"}}}`))
	}))
	defer srv.Close()

	type Attrs struct {
		Name string `json:"name"`
	}
	c := httpclient.NewWithTransport(srv.URL, "tok", &jsonapi.Transport{Wrapped: http.DefaultTransport})
	res, err := jsonapi.PatchSingle[Attrs](context.Background(), c, "/", map[string]any{})
	if err != nil {
		t.Fatalf("PatchSingle: %v", err)
	}
	if res.ID != "1" {
		t.Errorf("expected ID=1, got %q", res.ID)
	}
	if res.Type != "packages" {
		t.Errorf("expected Type=packages, got %q", res.Type)
	}
	if res.Attributes.Name != "mypkg" {
		t.Errorf("expected Attributes.Name=mypkg, got %q", res.Attributes.Name)
	}
}

// TestPatchSingle_ReturnsErrorOn4xx verifies PatchSingle surfaces JSON:API errors on 4xx.
func TestPatchSingle_ReturnsErrorOn4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"status":"422","detail":"name can't be blank"}]}`))
	}))
	defer srv.Close()

	type Attrs struct {
		Name string `json:"name"`
	}
	c := httpclient.NewWithTransport(srv.URL, "tok", &jsonapi.Transport{Wrapped: http.DefaultTransport})
	_, err := jsonapi.PatchSingle[Attrs](context.Background(), c, "/", map[string]any{})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !strings.Contains(err.Error(), "name can't be blank") {
		t.Errorf("expected error to contain 'name can\\'t be blank', got: %v", err)
	}
}
