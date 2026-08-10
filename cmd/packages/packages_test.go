package packages_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/cmd/packages"
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
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	}
	resp := m.responses[0]
	m.responses = m.responses[1:]
	return &http.Response{StatusCode: resp.status, Body: io.NopCloser(strings.NewReader(resp.body)), Header: make(http.Header)}, nil
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
	cmd := packages.NewCommand()
	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"packages","id":"pkg1","attributes":{"name":"mypackage","source_type":"helm"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "pkg1") {
		t.Errorf("expected pkg1 in output, got: %s", out.String())
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","attributes":{"name":"mypackage","source_type":"helm"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "pkg1") {
		t.Errorf("expected pkg1 in output, got: %s", out.String())
	}
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"packages","id":"pkg2","attributes":{"name":"newpkg","source_type":"helm"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "newpkg", "--source-type", "helm", "--source-url", "https://charts.example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out.String(), "pkg2") {
		t.Errorf("expected pkg2 in output, got: %s", out.String())
	}
}

// TestCreate_JSONAPIBodyEnvelope verifies the create request body is a JSON:API envelope.
func TestCreate_JSONAPIBodyEnvelope(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"packages","id":"pkg2","attributes":{"name":"newpkg","source_type":"helm"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "newpkg", "--source-type", "helm", "--source-url", "https://charts.example.com"}); err != nil {
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
	if data["type"] != "packages" {
		t.Errorf("expected data.type=packages, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["name"] != "newpkg" {
		t.Errorf("expected name=newpkg, got %v", attrs["name"])
	}
	if attrs["source_type"] != "helm" {
		t.Errorf("expected source_type=helm, got %v", attrs["source_type"])
	}
}

// TestCreate_JSONAPIContentType verifies create sends correct media types.
func TestCreate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"packages","id":"pkg2","attributes":{"name":"newpkg","source_type":"helm"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "newpkg", "--source-type", "helm", "--source-url", "https://charts.example.com"}); err != nil {
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

func TestUpdate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","links":{"self":"/api/v1/packages/pkg1"},"attributes":{"name":"mypackage","source_type":"helm"}}}`},
		{200, `{"data":{"type":"packages","id":"pkg1","attributes":{"name":"renamed","source_type":"git"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Errorf("expected renamed in output, got: %s", out.String())
	}
}

// TestUpdate_JSONAPIContentType verifies the update request sends correct media types.
func TestUpdate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","links":{"self":"/api/v1/packages/pkg1"},"attributes":{"name":"mypackage","source_type":"helm"}}}`},
		{200, `{"data":{"type":"packages","id":"pkg1","attributes":{"name":"renamed","source_type":"git"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) < 2 {
		t.Fatal("expected 2 HTTP calls (GET + PATCH)")
	}
	if got := mt.calls[1].Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got)
	}
	if got := mt.calls[1].Header.Get("Accept"); got != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", got)
	}
}

func TestUpdateNoFlags(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"pkg1"}); err == nil {
		t.Fatal("expected error when no flags provided")
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","links":{"self":"/api/v1/packages/pkg1"},"attributes":{"name":"mypackage","source_type":"helm"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// TestDelete_JSONAPIAcceptHeader verifies that delete sends the correct Accept header.
func TestDelete_JSONAPIAcceptHeader(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","links":{"self":"/api/v1/packages/pkg1"},"attributes":{"name":"mypackage","source_type":"helm"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mt.calls) < 2 {
		t.Fatal("expected 2 HTTP calls (GET + DELETE)")
	}
	if got := mt.calls[1].Header.Get("Accept"); got != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", got)
	}
}

func TestGet404(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{404, `{"message":"not found"}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"missing"}); err == nil {
		t.Fatal("expected error for 404")
	}
}

// TestListFollowsNextLinks verifies that list follows links.next across multiple pages.
func TestListFollowsNextLinks(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"packages","id":"pkg1","attributes":{"name":"pkg1","source_type":"helm"}}],"links":{"next":"/api/v1/packages?page=2"}}`},
		{200, `{"data":[{"type":"packages","id":"pkg2","attributes":{"name":"pkg2","source_type":"git"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(mt.calls) != 2 {
		t.Errorf("expected 2 HTTP calls (one per page), got %d", len(mt.calls))
	}
	if !strings.Contains(mt.calls[1].URL.RawQuery, "page=2") {
		t.Errorf("expected second call to use page=2 query, got: %s", mt.calls[1].URL.RawQuery)
	}
	if !strings.Contains(out.String(), "pkg1") {
		t.Errorf("expected pkg1 (page 1) in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "pkg2") {
		t.Errorf("expected pkg2 (page 2) in output, got: %s", out.String())
	}
}

// TestUpdateUsesSelfLink verifies that update uses data.links.self for the PATCH URL.
func TestUpdateUsesSelfLink(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","links":{"self":"/api/v1/packages/pkg1-canonical"},"attributes":{"name":"mypackage","source_type":"helm"}}}`},
		{200, `{"data":{"type":"packages","id":"pkg1","attributes":{"name":"renamed","source_type":"helm"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(mt.calls))
	}
	if mt.calls[0].Method != http.MethodGet {
		t.Errorf("expected first call to be GET, got %s", mt.calls[0].Method)
	}
	if mt.calls[1].Method != http.MethodPatch {
		t.Errorf("expected second call to be PATCH, got %s", mt.calls[1].Method)
	}
	if mt.calls[1].URL.Path != "/api/v1/packages/pkg1-canonical" {
		t.Errorf("expected PATCH to use self link /api/v1/packages/pkg1-canonical, got: %s", mt.calls[1].URL.Path)
	}
}

// TestDeleteUsesSelfLink verifies that delete uses data.links.self for the DELETE URL.
func TestDeleteUsesSelfLink(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"packages","id":"pkg1","links":{"self":"/api/v1/packages/pkg1-canonical"},"attributes":{"name":"mypackage","source_type":"helm"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := packages.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"pkg1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mt.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(mt.calls))
	}
	if mt.calls[1].Method != http.MethodDelete {
		t.Errorf("expected second call to be DELETE, got %s", mt.calls[1].Method)
	}
	if mt.calls[1].URL.Path != "/api/v1/packages/pkg1-canonical" {
		t.Errorf("expected DELETE to use self link /api/v1/packages/pkg1-canonical, got: %s", mt.calls[1].URL.Path)
	}
}
