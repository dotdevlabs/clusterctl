package deployments_test

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

	"github.com/dotdevlabs/clusterctl/cmd/deployments"
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
	cmd := deployments.NewCommand()
	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"deployed"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "d1") {
		t.Errorf("expected d1 in output, got: %s", out.String())
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"deployed"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "d1") {
		t.Errorf("expected d1 in output, got: %s", out.String())
	}
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d2","attributes":{"name":"my-deploy","project_id":"p1","cluster_id":"c1","namespace":"default","package_name":"promtail","status":"pending"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--cluster-id", "c1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "6.17.1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out.String(), "d2") {
		t.Errorf("expected d2 in output, got: %s", out.String())
	}
	if mt.calls[0].Method != http.MethodPost {
		t.Errorf("expected POST, got %s", mt.calls[0].Method)
	}
}

func TestCreate_RequestBodyShape(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d3","attributes":{"name":"host-log-collector","namespace":"default","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--cluster-id", "c1",
		"--name", "host-log-collector",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "6.17.1",
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
	if data["type"] != "deployments" {
		t.Errorf("expected data.type=deployments, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["name"] != "host-log-collector" {
		t.Errorf("expected name=host-log-collector, got %v", attrs["name"])
	}
	if attrs["namespace"] != "default" {
		t.Errorf("expected namespace=default, got %v", attrs["namespace"])
	}
	if attrs["package_name"] != "promtail" {
		t.Errorf("expected package_name=promtail, got %v", attrs["package_name"])
	}
	if attrs["package_version"] != "6.17.1" {
		t.Errorf("expected package_version=6.17.1, got %v", attrs["package_version"])
	}
	if _, ok := attrs["package_id"]; ok {
		t.Errorf("expected package_id to be absent, but it was present: %v", attrs["package_id"])
	}
}

// TestCreate_JSONAPIContentType verifies create sends correct media types.
func TestCreate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d2","attributes":{"name":"my-deploy","project_id":"p1","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
	}); err != nil {
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
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","project_id":"p1","cluster_id":"c1","package_name":"promtail","package_version":"2.0.0","status":"deployed"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--package-version", "2.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "d1") {
		t.Errorf("expected d1 in output, got: %s", out.String())
	}
}

func TestUpdate_RequestBodyShape(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","namespace":"kube-system","package_name":"promtail-new","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--package-name", "promtail-new", "--namespace", "kube-system"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) < 2 {
		t.Fatal("expected 2 HTTP calls (GET + PATCH)")
	}
	raw, err := io.ReadAll(mt.calls[1].Body)
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
	if data["type"] != "deployments" {
		t.Errorf("expected data.type=deployments, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["package_name"] != "promtail-new" {
		t.Errorf("expected package_name=promtail-new, got %v", attrs["package_name"])
	}
	if attrs["namespace"] != "kube-system" {
		t.Errorf("expected namespace=kube-system, got %v", attrs["namespace"])
	}
	if _, ok := attrs["package_id"]; ok {
		t.Errorf("expected package_id to be absent, but it was present: %v", attrs["package_id"])
	}
}

// TestUpdate_JSONAPIContentType verifies the update request sends correct media types.
func TestUpdate_JSONAPIContentType(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--package-version", "2.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
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

func TestCreate422WithError(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{422, `{"error":"validation failed: name can't be blank"}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--cluster-id", "c1",
		"--name", "x",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected error to contain 'validation failed', got: %s", err.Error())
	}
}

func TestUpdateNoFlags(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"d1"}); err == nil {
		t.Fatal("expected error when no flags provided")
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// TestDelete_JSONAPIAcceptHeader verifies that delete sends the correct Accept header.
func TestDelete_JSONAPIAcceptHeader(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
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
	parent := deployments.NewCommand()
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
		{200, `{"data":[{"type":"deployments","id":"d1","attributes":{"name":"deploy1","project_id":"p1","status":"deployed"}}],"links":{"next":"/api/v1/deployments?page=2"}}`},
		{200, `{"data":[{"type":"deployments","id":"d2","attributes":{"name":"deploy2","project_id":"p1","status":"pending"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
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
	if !strings.Contains(out.String(), "d1") {
		t.Errorf("expected d1 (page 1) in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "d2") {
		t.Errorf("expected d2 (page 2) in output, got: %s", out.String())
	}
}

// TestUpdateUsesSelfLink verifies that update uses data.links.self for the PATCH URL.
func TestUpdateUsesSelfLink(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1-canonical"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","project_id":"p1","package_version":"2.0.0","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--package-version", "2.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
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
	if mt.calls[1].URL.Path != "/api/v1/deployments/d1-canonical" {
		t.Errorf("expected PATCH to use self link /api/v1/deployments/d1-canonical, got: %s", mt.calls[1].URL.Path)
	}
}

// TestDeleteUsesSelfLink verifies that delete uses data.links.self for the DELETE URL.
func TestDeleteUsesSelfLink(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1-canonical"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mt.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(mt.calls))
	}
	if mt.calls[1].Method != http.MethodDelete {
		t.Errorf("expected second call to be DELETE, got %s", mt.calls[1].Method)
	}
	if mt.calls[1].URL.Path != "/api/v1/deployments/d1-canonical" {
		t.Errorf("expected DELETE to use self link /api/v1/deployments/d1-canonical, got: %s", mt.calls[1].URL.Path)
	}
}

func TestCreate_TemplateExtraResources(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d10","attributes":{"name":"my-deploy","project_id":"p1","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--template-extra-resources", `{"extra.yaml":"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: extra"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	raw, _ := io.ReadAll(mt.calls[0].Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	if data["type"] != "deployments" {
		t.Errorf("expected data.type=deployments, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	ter, ok := attrs["template_extra_resources"]
	if !ok {
		t.Fatal("expected template_extra_resources in body attributes")
	}
	terMap, ok := ter.(map[string]any)
	if !ok {
		t.Fatalf("expected template_extra_resources to be an object, got %T", ter)
	}
	if _, ok := terMap["extra.yaml"]; !ok {
		t.Error("expected extra.yaml key in template_extra_resources")
	}
	if _, ok := attrs["values_override"]; ok {
		t.Error("values_override should be absent when not passed")
	}
}

func TestUpdate_TemplateExtraResources(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--template-extra-resources", `{"patch.yaml":"apiVersion: v1\nkind: ConfigMap"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) < 2 {
		t.Fatal("expected 2 HTTP calls (GET + PATCH)")
	}
	raw, _ := io.ReadAll(mt.calls[1].Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if _, ok := attrs["template_extra_resources"]; !ok {
		t.Error("expected template_extra_resources in PATCH body")
	}
}

func TestCreate_NewScalarFields(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d20","attributes":{"name":"my-deploy","project_id":"p1","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--is-ai",
		"--min-replicas", "2",
		"--max-replicas", "10",
		"--scaling-mode", "hpa",
		"--canary-enabled",
		"--canary-step-weight", "20",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, _ := io.ReadAll(mt.calls[0].Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["is_ai"] != true {
		t.Errorf("expected is_ai=true, got %v", attrs["is_ai"])
	}
	if attrs["min_replicas"] != float64(2) {
		t.Errorf("expected min_replicas=2, got %v", attrs["min_replicas"])
	}
	if attrs["max_replicas"] != float64(10) {
		t.Errorf("expected max_replicas=10, got %v", attrs["max_replicas"])
	}
	if attrs["scaling_mode"] != "hpa" {
		t.Errorf("expected scaling_mode=hpa, got %v", attrs["scaling_mode"])
	}
	if attrs["canary_enabled"] != true {
		t.Errorf("expected canary_enabled=true, got %v", attrs["canary_enabled"])
	}
	if attrs["canary_step_weight"] != float64(20) {
		t.Errorf("expected canary_step_weight=20, got %v", attrs["canary_step_weight"])
	}
}

func TestCreate_NodeSelector(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d30","attributes":{"name":"my-deploy","project_id":"p1","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--node-selector", `{"kubernetes.io/os":"linux"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, _ := io.ReadAll(mt.calls[0].Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bodyData, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	attrs, ok := bodyData["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", bodyData["attributes"])
	}
	ns, ok := attrs["node_selector"]
	if !ok {
		t.Fatal("expected node_selector in body")
	}
	nsMap, ok := ns.(map[string]any)
	if !ok {
		t.Fatalf("expected node_selector to be object, got %T", ns)
	}
	if nsMap["kubernetes.io/os"] != "linux" {
		t.Errorf("expected kubernetes.io/os=linux, got %v", nsMap["kubernetes.io/os"])
	}
}

func TestCreate_Tolerations(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"deployments","id":"d40","attributes":{"name":"my-deploy","project_id":"p1","status":"pending"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--tolerations", `[{"key":"dedicated","operator":"Equal","value":"gpu","effect":"NoSchedule"}]`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, _ := io.ReadAll(mt.calls[0].Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bodyData, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	attrs, ok := bodyData["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", bodyData["attributes"])
	}
	tols, ok := attrs["tolerations"]
	if !ok {
		t.Fatal("expected tolerations in body")
	}
	tolsList, ok := tols.([]any)
	if !ok {
		t.Fatalf("expected tolerations to be array, got %T", tols)
	}
	if len(tolsList) != 1 {
		t.Fatalf("expected 1 toleration, got %d", len(tolsList))
	}
	tol, ok := tolsList[0].(map[string]any)
	if !ok {
		t.Fatalf("expected toleration to be object, got %T", tolsList[0])
	}
	if tol["key"] != "dedicated" {
		t.Errorf("expected key=dedicated, got %v", tol["key"])
	}
	if tol["effect"] != "NoSchedule" {
		t.Errorf("expected effect=NoSchedule, got %v", tol["effect"])
	}
}

func TestCreate_InvalidNodeSelectorJSON(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--node-selector", "not json",
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for invalid node-selector JSON")
	}
	if !strings.Contains(err.Error(), "node-selector") {
		t.Errorf("expected error to mention node-selector, got: %s", err.Error())
	}
	if len(mt.calls) > 0 {
		t.Error("expected no HTTP calls on validation failure")
	}
}

func TestCreate_InvalidTolerationsJSON(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--tolerations", "not json",
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for invalid tolerations JSON")
	}
	if len(mt.calls) > 0 {
		t.Error("expected no HTTP calls on validation failure")
	}
}

func TestCreate_InvalidTemplateExtraResourcesJSON(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "my-deploy",
		"--namespace", "default",
		"--package-name", "promtail",
		"--package-version", "1.0.0",
		"--template-extra-resources", `"just a string"`,
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for wrong type (string instead of object)")
	}
	if !strings.Contains(err.Error(), "template-extra-resources") {
		t.Errorf("expected error to mention template-extra-resources, got: %s", err.Error())
	}
	if len(mt.calls) > 0 {
		t.Error("expected no HTTP calls on validation failure")
	}
}

func TestGet_DecodesNewFields(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d50","attributes":{"name":"ai-deploy","project_id":"p1","status":"deployed","is_ai":true,"min_replicas":3,"template_extra_resources":{"file.yaml":"apiVersion: v1\nkind: ConfigMap"}}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"d50"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	outStr := out.String()
	if !strings.Contains(outStr, "d50") {
		t.Errorf("expected d50 in output, got: %s", outStr)
	}
	// Verify the JSON output decodes the new fields correctly
	var env struct {
		Data struct {
			IsAI                   *bool             `json:"is_ai"`
			MinReplicas            *int              `json:"min_replicas"`
			TemplateExtraResources map[string]string `json:"template_extra_resources"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(outStr), &env); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if env.Data.IsAI == nil || !*env.Data.IsAI {
		t.Errorf("expected is_ai=true in decoded output, got: %v", env.Data.IsAI)
	}
	if env.Data.MinReplicas == nil || *env.Data.MinReplicas != 3 {
		t.Errorf("expected min_replicas=3, got: %v", env.Data.MinReplicas)
	}
	if env.Data.TemplateExtraResources == nil {
		t.Error("expected template_extra_resources in decoded output")
	} else if _, ok := env.Data.TemplateExtraResources["file.yaml"]; !ok {
		t.Error("expected file.yaml key in template_extra_resources")
	}
}

func TestUpdate_NewScalarFields(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"my-deploy","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"my-deploy","project_id":"p1","status":"deployed","min_replicas":3,"canary_enabled":true}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--min-replicas", "3", "--canary-enabled"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) < 2 {
		t.Fatal("expected 2 HTTP calls (GET + PATCH)")
	}
	raw, _ := io.ReadAll(mt.calls[1].Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be an object, got: %T", body["data"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["min_replicas"] != float64(3) {
		t.Errorf("expected min_replicas=3, got %v", attrs["min_replicas"])
	}
	if attrs["canary_enabled"] != true {
		t.Errorf("expected canary_enabled=true, got %v", attrs["canary_enabled"])
	}
	// Verify only changed fields are sent (no zero-value pollution)
	for k := range attrs {
		if k != "min_replicas" && k != "canary_enabled" {
			t.Errorf("unexpected field %q in PATCH body (only changed fields should be sent)", k)
		}
	}
}
