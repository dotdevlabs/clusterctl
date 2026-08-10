package projects_test

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

	"github.com/dotdevlabs/clusterctl/cmd/projects"
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
	cmd := projects.NewCommand()
	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"projects","id":"p1","attributes":{"name":"myproject"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "p1") {
		t.Errorf("expected p1 in output, got: %s", out.String())
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"projects","id":"p1","attributes":{"name":"myproject"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "p1") {
		t.Errorf("expected p1 in output, got: %s", out.String())
	}
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"projects","id":"p2","attributes":{"name":"newproject"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "newproject"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out.String(), "p2") {
		t.Errorf("expected p2 in output, got: %s", out.String())
	}
}

// TestCreate_JSONAPIBodyEnvelope verifies the create request body is a JSON:API envelope.
func TestCreate_JSONAPIBodyEnvelope(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"projects","id":"p2","attributes":{"name":"newproject"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "newproject"}); err != nil {
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
	if data["type"] != "projects" {
		t.Errorf("expected data.type=projects, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["name"] != "newproject" {
		t.Errorf("expected name=newproject, got %v", attrs["name"])
	}
}

// TestCreate_JSONAPIContentType verifies the create request sends correct media types.
func TestCreate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"projects","id":"p2","attributes":{"name":"newproject"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "newproject"}); err != nil {
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
		{200, `{"data":{"type":"projects","id":"p1","links":{"self":"/api/v1/projects/p1"},"attributes":{"name":"myproject"}}}`},
		{200, `{"data":{"type":"projects","id":"p1","attributes":{"name":"renamed"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Errorf("expected renamed in output, got: %s", out.String())
	}
}

// TestUpdate_JSONAPIContentType verifies the update request sends correct media types.
func TestUpdate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"projects","id":"p1","links":{"self":"/api/v1/projects/p1"},"attributes":{"name":"myproject"}}}`},
		{200, `{"data":{"type":"projects","id":"p1","attributes":{"name":"renamed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
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
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	err = sub.RunE(sub, []string{"p1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"projects","id":"p1","links":{"self":"/api/v1/projects/p1"},"attributes":{"name":"myproject"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// TestDelete_JSONAPIAcceptHeader verifies that delete sends the correct Accept header.
func TestDelete_JSONAPIAcceptHeader(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"projects","id":"p1","links":{"self":"/api/v1/projects/p1"},"attributes":{"name":"myproject"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
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
	parent := projects.NewCommand()
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
		{200, `{"data":[{"type":"projects","id":"p1","attributes":{"name":"proj1"}}],"links":{"next":"/api/v1/projects?page=2"}}`},
		{200, `{"data":[{"type":"projects","id":"p2","attributes":{"name":"proj2"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := projects.NewCommand()
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
	if !strings.Contains(out.String(), "p1") {
		t.Errorf("expected p1 (page 1) in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "p2") {
		t.Errorf("expected p2 (page 2) in output, got: %s", out.String())
	}
}

// TestUpdateUsesSelfLink verifies that update uses data.links.self for the PATCH URL.
func TestUpdateUsesSelfLink(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"projects","id":"p1","links":{"self":"/api/v1/projects/p1-canonical"},"attributes":{"name":"myproject"}}}`},
		{200, `{"data":{"type":"projects","id":"p1","attributes":{"name":"renamed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
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
	if mt.calls[1].URL.Path != "/api/v1/projects/p1-canonical" {
		t.Errorf("expected PATCH to use self link /api/v1/projects/p1-canonical, got: %s", mt.calls[1].URL.Path)
	}
}

// TestDeleteUsesSelfLink verifies that delete uses data.links.self for the DELETE URL.
func TestDeleteUsesSelfLink(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"projects","id":"p1","links":{"self":"/api/v1/projects/p1-canonical"},"attributes":{"name":"myproject"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := projects.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mt.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(mt.calls))
	}
	if mt.calls[1].Method != http.MethodDelete {
		t.Errorf("expected second call to be DELETE, got %s", mt.calls[1].Method)
	}
	if mt.calls[1].URL.Path != "/api/v1/projects/p1-canonical" {
		t.Errorf("expected DELETE to use self link /api/v1/projects/p1-canonical, got: %s", mt.calls[1].URL.Path)
	}
}
