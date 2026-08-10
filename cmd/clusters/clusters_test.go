package clusters_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/cmd/clusters"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

type mockTransport struct {
	responses []mockResponse
	calls     []*http.Request
}

type mockResponse struct {
	status int
	body   string
}

func (m *mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, r)
	if len(m.responses) == 0 {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Header:     make(http.Header),
		}, nil
	}
	resp := m.responses[0]
	m.responses = m.responses[1:]
	return &http.Response{
		StatusCode: resp.status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Header:     make(http.Header),
	}, nil
}

func buildCtx(t *testing.T, transport http.RoundTripper, jsonMode bool) (context.Context, *bytes.Buffer) {
	t.Helper()
	var out, errOut bytes.Buffer
	client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: transport})
	renderer := output.New(jsonMode, "", &out, &errOut)
	ctx := context.Background()
	ctx = ctxutil.WithClient(ctx, client)
	ctx = ctxutil.WithRenderer(ctx, renderer)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: jsonMode})
	return ctx, &out
}

func TestNewCommand(t *testing.T) {
	cmd := clusters.NewCommand()
	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}
	if cmd.Use != "clusters" {
		t.Errorf("expected Use=clusters, got %q", cmd.Use)
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"clusters","id":"c1","attributes":{"name":"prod","cluster_type":"virtual","status":"ready"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "c1") {
		t.Errorf("expected c1 in output, got: %s", out.String())
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"clusters","id":"c1","attributes":{"name":"prod","cluster_type":"virtual","status":"ready"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "c1") {
		t.Errorf("expected c1 in output, got: %s", out.String())
	}
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"clusters","id":"c2","attributes":{"name":"dev","cluster_type":"virtual","status":"pending"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "dev", "--cluster-type", "virtual"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out.String(), "c2") {
		t.Errorf("expected c2 in output, got: %s", out.String())
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if mt.calls[0].Method != http.MethodPost {
		t.Errorf("expected POST, got %s", mt.calls[0].Method)
	}
}

// TestCreate_JSONAPIBodyEnvelope verifies the create request body is a JSON:API envelope.
func TestCreate_JSONAPIBodyEnvelope(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"clusters","id":"c2","attributes":{"name":"dev","cluster_type":"virtual","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "dev", "--cluster-type", "virtual"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	raw, err := io.ReadAll(mt.calls[0].Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	if data["type"] != "clusters" {
		t.Errorf("expected data.type=clusters, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["cluster_type"] != "virtual" {
		t.Errorf("expected cluster_type=virtual, got %v", attrs["cluster_type"])
	}
}

// TestCreate_JSONAPIContentType verifies the create request sends correct media types.
func TestCreate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"clusters","id":"c2","attributes":{"name":"dev","cluster_type":"virtual","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "dev", "--cluster-type", "virtual"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if got := mt.calls[0].Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got)
	}
	if got := mt.calls[0].Header.Get("Accept"); got != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", got)
	}
}

func TestCreateDryRun(t *testing.T) {
	mt := &mockTransport{}
	ctx, out := buildCtx(t, mt, false)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{DryRun: true})
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{"--name", "dev", "--cluster-type", "virtual"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	if len(mt.calls) != 0 {
		t.Error("expected no HTTP calls in dry-run mode")
	}
	if !strings.Contains(out.String(), "virtual") {
		t.Errorf("expected dry-run body in output, got: %s", out.String())
	}
}

func TestUpdate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"clusters","id":"c1","attributes":{"name":"prod","cluster_type":"virtual","status":"ready"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--k8s-base-hostname", "example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "c1") {
		t.Errorf("expected c1 in output, got: %s", out.String())
	}
	if mt.calls[0].Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", mt.calls[0].Method)
	}
}

// TestUpdate_JSONAPIBodyEnvelope verifies the update request body is a JSON:API envelope.
func TestUpdate_JSONAPIBodyEnvelope(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"clusters","id":"c1","attributes":{"name":"prod","cluster_type":"virtual","status":"ready"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--k8s-base-hostname", "example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	raw, err := io.ReadAll(mt.calls[0].Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	if data["type"] != "clusters" {
		t.Errorf("expected data.type=clusters, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["k8s_base_hostname"] != "example.com" {
		t.Errorf("expected k8s_base_hostname=example.com, got %v", attrs["k8s_base_hostname"])
	}
}

// TestUpdate_JSONAPIContentType verifies the update request sends correct media types.
func TestUpdate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"clusters","id":"c1","attributes":{"name":"prod","cluster_type":"virtual","status":"ready"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--k8s-base-hostname", "example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if got := mt.calls[0].Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got)
	}
	if got := mt.calls[0].Header.Get("Accept"); got != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", got)
	}
}

func TestUpdateNoFlags(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	err = sub.RunE(sub, []string{"c1"})
	if err == nil {
		t.Fatal("expected error when no flags provided")
	}
	if !strings.Contains(err.Error(), "at least one flag") {
		t.Errorf("expected 'at least one flag' error, got: %v", err)
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if mt.calls[0].Method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", mt.calls[0].Method)
	}
}

func TestHealthCheck(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{202, `{"data":{"type":"health_checks","id":"1","attributes":{"status":"health_check_enqueued"}}}`},
	}}
	ctx, out := buildCtx(t, mt, false)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"health-check"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("health-check: %v", err)
	}
	if !strings.Contains(out.String(), "health_check_enqueued") {
		t.Errorf("expected health_check_enqueued in output, got: %s", out.String())
	}
}

func TestFluxBootstrap(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{202, `{"data":{"type":"flux_bootstraps","id":"1","attributes":{"name":"prod","flux_bootstrap_status":"bootstrapped"}}}`},
	}}
	ctx, out := buildCtx(t, mt, false)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"flux-bootstrap"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("flux-bootstrap: %v", err)
	}
	if !strings.Contains(out.String(), "bootstrapped") {
		t.Errorf("expected bootstrapped in output, got: %s", out.String())
	}
}

func TestGet404(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{404, `{"message":"not found"}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	err = sub.RunE(sub, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func buildCtxFromURL(t *testing.T, baseURL string, jsonMode bool) (context.Context, *bytes.Buffer) {
	t.Helper()
	var out, errOut bytes.Buffer
	client := httpclient.New(baseURL, "tok")
	renderer := output.New(jsonMode, "", &out, &errOut)
	ctx := context.Background()
	ctx = ctxutil.WithClient(ctx, client)
	ctx = ctxutil.WithRenderer(ctx, renderer)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: jsonMode})
	return ctx, &out
}

// TestCreate_JSONAPIErrorSurfacing verifies JSON:API errors[] are surfaced on 4xx.
func TestCreate_JSONAPIErrorSurfacing(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{400, `{"errors":[{"status":"400","detail":"cluster_type is invalid"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "bad", "--cluster-type", "bad"}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "cluster_type is invalid") {
		t.Errorf("expected error to contain 'cluster_type is invalid', got: %v", err)
	}
}

func TestGetPopulatesNonIDAttributes(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"clusters","id":"c99","attributes":{"name":"prod-cluster","cluster_type":"virtual","status":"ready"}}}`))
	}))
	defer srv.Close()

	ctx, out := buildCtxFromURL(t, srv.URL, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"c99"}); err != nil {
		t.Fatalf("get: %v", err)
	}

	if gotAccept != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want %q", gotAccept, "application/vnd.api+json")
	}
	if !strings.Contains(out.String(), "prod-cluster") {
		t.Errorf("expected name attribute in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "virtual") {
		t.Errorf("expected cluster_type attribute in output, got: %s", out.String())
	}
}
