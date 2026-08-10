package secrets_test

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

	"github.com/dotdevlabs/clusterctl/cmd/secrets"
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
	cmd := secrets.NewCommand()
	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"project_secrets","id":"s1","attributes":{"kubernetes_secret_name":"app-secrets","key":"DATABASE_URL"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "s1") {
		t.Errorf("expected s1 in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "app-secrets") {
		t.Errorf("expected app-secrets in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "DATABASE_URL") {
		t.Errorf("expected DATABASE_URL in output, got: %s", out.String())
	}
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"project_secrets","id":"s2","attributes":{"kubernetes_secret_name":"app-secrets","key":"SECRET_KEY_BASE"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--secret-name", "app-secrets", "--key", "SECRET_KEY_BASE", "--value", "myvalue"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out.String(), "s2") {
		t.Errorf("expected s2 in output, got: %s", out.String())
	}
	if mt.calls[0].Method != http.MethodPost {
		t.Errorf("expected POST, got %s", mt.calls[0].Method)
	}
}

func TestCreate_RequestBodyShape(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"project_secrets","id":"s3","attributes":{"kubernetes_secret_name":"app-secrets","key":"DATABASE_URL"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--secret-name", "app-secrets",
		"--key", "DATABASE_URL",
		"--value", "supersecret",
	}); err != nil {
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
	if data["type"] != "project_secrets" {
		t.Errorf("expected data.type=project_secrets, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["kubernetes_secret_name"] != "app-secrets" {
		t.Errorf("expected kubernetes_secret_name=app-secrets, got %v", attrs["kubernetes_secret_name"])
	}
	if attrs["key"] != "DATABASE_URL" {
		t.Errorf("expected key=DATABASE_URL, got %v", attrs["key"])
	}
	if attrs["value"] != "supersecret" {
		t.Errorf("expected value=supersecret, got %v", attrs["value"])
	}
}

// TestCreate_JSONAPIContentType verifies that create sends the correct Content-Type.
func TestCreate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"project_secrets","id":"s4","attributes":{"kubernetes_secret_name":"app-secrets","key":"K"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--secret-name", "app-secrets", "--key", "K", "--value", "v"}); err != nil {
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

// TestDelete_JSONAPIAcceptHeader verifies that delete sends the correct Accept header.
func TestDelete_JSONAPIAcceptHeader(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"s1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if got := mt.calls[0].Header.Get("Accept"); got != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", got)
	}
}

func TestList_ShowsKubernetesSecretNameAndKey(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[` +
			`{"type":"project_secrets","id":"s1","attributes":{"kubernetes_secret_name":"app-secrets","key":"DATABASE_URL"}},` +
			`{"type":"project_secrets","id":"s2","attributes":{"kubernetes_secret_name":"app-secrets","key":"CACHE_DATABASE_URL"}}` +
			`],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "DATABASE_URL") {
		t.Errorf("expected DATABASE_URL in output, got: %s", got)
	}
	if !strings.Contains(got, "CACHE_DATABASE_URL") {
		t.Errorf("expected CACHE_DATABASE_URL in output, got: %s", got)
	}
	if strings.Count(got, "app-secrets") < 1 {
		t.Errorf("expected app-secrets in output, got: %s", got)
	}
}

func TestCreateDryRun(t *testing.T) {
	mt := &mockTransport{}
	var out, errOut bytes.Buffer
	client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: mt})
	renderer := output.New(true, "", &out, &errOut)
	ctx := context.Background()
	ctx = ctxutil.WithClient(ctx, client)
	ctx = ctxutil.WithRenderer(ctx, renderer)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})

	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(&out)
	if err := sub.ParseFlags([]string{
		"--secret-name", "app-secrets",
		"--key", "DATABASE_URL",
		"--value", "dryval",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("dry-run create: %v", err)
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected no HTTP calls in dry-run, got %d", len(mt.calls))
	}
	got := out.String()
	if !strings.Contains(got, "kubernetes_secret_name") {
		t.Errorf("expected kubernetes_secret_name in dry-run output, got: %s", got)
	}
	if !strings.Contains(got, "project_secrets") {
		t.Errorf("expected project_secrets resource type in dry-run output, got: %s", got)
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"s1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if mt.calls[0].Method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", mt.calls[0].Method)
	}
	if !strings.Contains(mt.calls[0].URL.Path, "/secrets/s1") {
		t.Errorf("expected /secrets/s1 in path, got: %s", mt.calls[0].URL.Path)
	}
}

func TestMaterialize(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"secret_materializations","id":null,"attributes":{"applied_count":3,"message":"materialized"}}}`},
	}}
	ctx, out := buildCtx(t, mt, false)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"materialize"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if !strings.Contains(out.String(), "materialized") {
		t.Errorf("expected materialized in output, got: %s", out.String())
	}
	if !strings.Contains(mt.calls[0].URL.Path, "secret_materialization") {
		t.Errorf("expected secret_materialization in path, got: %s", mt.calls[0].URL.Path)
	}
}

// TestCreate_JSONAPIErrorSurfacing verifies JSON:API errors[] are surfaced on 4xx.
func TestCreate_JSONAPIErrorSurfacing(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{422, `{"errors":[{"status":"422","detail":"key has already been taken"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := secrets.NewCommand()
	if err := parent.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatal(err)
	}
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--secret-name", "app-secrets", "--key", "DATABASE_URL", "--value", "v"}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !strings.Contains(err.Error(), "key has already been taken") {
		t.Errorf("expected error to contain 'key has already been taken', got: %v", err)
	}
}

func TestListMissingProjectID(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := secrets.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err == nil {
		t.Fatal("expected error when --project-id missing")
	}
}
