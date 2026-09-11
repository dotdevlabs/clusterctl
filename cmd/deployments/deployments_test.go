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

// mustAttrs extracts data.attributes from a JSON body map, fataling on bad shape.
func mustAttrs(t *testing.T, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected body.data to be an object, got %T", body["data"])
	}
	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got %T", data["attributes"])
	}
	return attrs
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

// TestCreate_AllScalarFields verifies dry-run create with all scalar flags produces a body with those fields.
func TestCreate_AllScalarFields(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "2.0.0",
		"--cluster-id", "c1",
		"--environment-preset", "prod",
		"--values-override", "key: val",
		"--scaling-mode", "hpa",
		"--cpu-request", "100m",
		"--cpu-limit", "500m",
		"--memory-request", "128Mi",
		"--memory-limit", "512Mi",
		"--scaling-profile", "balanced",
		"--placement-policy", "spread",
		"--canary-interval", "1m",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	for _, field := range []string{"project_id", "name", "namespace", "package_name", "package_version", "cluster_id", "environment_preset", "values_override", "scaling_mode", "cpu_request", "cpu_limit", "memory_request", "memory_limit", "scaling_profile", "placement_policy", "canary_interval"} {
		if _, ok := attrs[field]; !ok {
			t.Errorf("missing field %q in dry-run body", field)
		}
	}
}

// TestCreate_TemplateExtraResources verifies dry-run create includes template_extra_resources.
func TestCreate_TemplateExtraResources(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--template-extra-resources", `{"config.yaml":"content here"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	ter, ok := attrs["template_extra_resources"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected template_extra_resources to be object, got %T: %v", attrs["template_extra_resources"], attrs["template_extra_resources"])
	}
	if ter["config.yaml"] != "content here" {
		t.Errorf("expected config.yaml=content here, got %v", ter["config.yaml"])
	}
}

// TestCreate_TemplateValues verifies dry-run create includes template_values.
func TestCreate_TemplateValues(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--template-values", `{"replicas":"3"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	tv, ok := attrs["template_values"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected template_values to be object, got %T: %v", attrs["template_values"], attrs["template_values"])
	}
	if tv["replicas"] != "3" {
		t.Errorf("expected replicas=3, got %v", tv["replicas"])
	}
}

// TestCreate_NodeSelector verifies dry-run create includes node_selector.
func TestCreate_NodeSelector(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--node-selector", `{"tier":"frontend"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	ns, ok := attrs["node_selector"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected node_selector to be object, got %T", attrs["node_selector"])
	}
	if ns["tier"] != "frontend" {
		t.Errorf("expected tier=frontend, got %v", ns["tier"])
	}
}

// TestCreate_Tolerations verifies dry-run create includes tolerations array.
func TestCreate_Tolerations(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--tolerations", `[{"key":"spot","operator":"Exists","effect":"NoSchedule"}]`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	tols, ok := attrs["tolerations"].([]interface{})
	if !ok || len(tols) == 0 {
		t.Fatalf("expected tolerations to be non-empty array, got %T: %v", attrs["tolerations"], attrs["tolerations"])
	}
	tol, ok2 := tols[0].(map[string]interface{})
	if !ok2 {
		t.Fatalf("expected toleration element to be object, got %T", tols[0])
	}
	if tol["key"] != "spot" || tol["effect"] != "NoSchedule" {
		t.Errorf("unexpected toleration values: %v", tol)
	}
}

// TestCreate_MalformedNodeSelector verifies that invalid node-selector JSON returns an error.
func TestCreate_MalformedNodeSelector(t *testing.T) {
	ctx, _ := buildCtx(t, &mockTransport{}, false)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--node-selector", "not-json",
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for malformed node-selector")
	}
	if !strings.Contains(err.Error(), "node-selector") {
		t.Errorf("expected error to mention node-selector, got: %v", err)
	}
}

// TestCreate_MalformedTolerations verifies that non-array JSON for tolerations returns an error.
func TestCreate_MalformedTolerations(t *testing.T) {
	ctx, _ := buildCtx(t, &mockTransport{}, false)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--tolerations", `{"key":"v"}`,
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for non-array tolerations")
	}
	if !strings.Contains(err.Error(), "tolerations") {
		t.Errorf("expected error to mention tolerations, got: %v", err)
	}
}

// TestCreate_MalformedTemplateExtraResources verifies non-string values are rejected.
func TestCreate_MalformedTemplateExtraResources(t *testing.T) {
	ctx, _ := buildCtx(t, &mockTransport{}, false)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--template-extra-resources", `{"file.yaml": 123}`,
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for non-string template-extra-resources value")
	}
	if !strings.Contains(err.Error(), "template-extra-resources") {
		t.Errorf("expected error to mention template-extra-resources, got: %v", err)
	}
}

// TestCreate_MalformedTemplateValues verifies that a non-object JSON value is rejected.
func TestCreate_MalformedTemplateValues(t *testing.T) {
	ctx, _ := buildCtx(t, &mockTransport{}, false)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--template-values", `[1,2,3]`,
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for non-object template-values")
	}
	if !strings.Contains(err.Error(), "template-values") {
		t.Errorf("expected error to mention template-values, got: %v", err)
	}
}

// TestCreate_IntegerFlags verifies integer flags appear in dry-run body.
func TestCreate_IntegerFlags(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--min-replicas", "3",
		"--max-replicas", "10",
		"--desired-replicas", "5",
		"--cpu-target-utilization", "80",
		"--memory-target-utilization", "75",
		"--canary-step-weight", "10",
		"--canary-max-weight", "50",
		"--canary-latency-p99-ms", "500",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	if attrs["min_replicas"] != float64(3) {
		t.Errorf("expected min_replicas=3, got %v", attrs["min_replicas"])
	}
	if attrs["max_replicas"] != float64(10) {
		t.Errorf("expected max_replicas=10, got %v", attrs["max_replicas"])
	}
	if attrs["canary_latency_p99_ms"] != float64(500) {
		t.Errorf("expected canary_latency_p99_ms=500, got %v", attrs["canary_latency_p99_ms"])
	}
}

// TestCreate_BooleanFlagTrue verifies bool flags appear in body when set.
func TestCreate_BooleanFlagTrue(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--is-ai",
		"--canary-enabled",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	if attrs["is_ai"] != true {
		t.Errorf("expected is_ai=true, got %v", attrs["is_ai"])
	}
	if attrs["canary_enabled"] != true {
		t.Errorf("expected canary_enabled=true, got %v", attrs["canary_enabled"])
	}
}

// TestCreate_BooleanFlagNotSet verifies bool flags are omitted when not provided.
func TestCreate_BooleanFlagNotSet(t *testing.T) {
	ctx, out := buildCtx(t, &mockTransport{}, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	if _, ok := attrs["is_ai"]; ok {
		t.Errorf("expected is_ai to be absent when not set, but it was present: %v", attrs["is_ai"])
	}
	if _, ok := attrs["canary_enabled"]; ok {
		t.Errorf("expected canary_enabled to be absent when not set, but it was present: %v", attrs["canary_enabled"])
	}
}

// TestCreate_DryRunNoHTTP verifies dry-run emits JSON and makes no HTTP calls.
func TestCreate_DryRunNoHTTP(t *testing.T) {
	mt := &mockTransport{}
	ctx, out := buildCtx(t, mt, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{
		"--project-id", "p1",
		"--name", "svc",
		"--namespace", "ns",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected no HTTP calls in dry-run, got %d", len(mt.calls))
	}
	if !strings.Contains(out.String(), "deployments") {
		t.Errorf("expected JSON output, got: %s", out.String())
	}
}

// TestUpdate_AllScalarFields verifies update sends only changed scalar fields.
func TestUpdate_AllScalarFields(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"svc","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"svc","project_id":"p1","cpu_request":"200m","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--cpu-request", "200m",
		"--memory-limit", "1Gi",
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	raw, err := io.ReadAll(mt.calls[1].Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	if attrs["cpu_request"] != "200m" {
		t.Errorf("expected cpu_request=200m, got %v", attrs["cpu_request"])
	}
	if attrs["memory_limit"] != "1Gi" {
		t.Errorf("expected memory_limit=1Gi, got %v", attrs["memory_limit"])
	}
	if _, ok := attrs["name"]; ok {
		t.Errorf("expected name to be absent (not changed), but present: %v", attrs["name"])
	}
}

// TestUpdate_TemplateExtraResources verifies update with template-extra-resources.
func TestUpdate_TemplateExtraResources(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"svc","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"svc","project_id":"p1","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--template-extra-resources", `{"f.yaml":"c"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	raw, err := io.ReadAll(mt.calls[1].Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	ter, ok := attrs["template_extra_resources"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected template_extra_resources object, got %T", attrs["template_extra_resources"])
	}
	if ter["f.yaml"] != "c" {
		t.Errorf("expected f.yaml=c, got %v", ter["f.yaml"])
	}
}

// TestUpdate_StructuredFieldsChangedOnly verifies only explicitly-changed flags go into PATCH body.
func TestUpdate_StructuredFieldsChangedOnly(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"deployments","id":"d1","links":{"self":"/api/v1/deployments/d1"},"attributes":{"name":"svc","project_id":"p1","status":"deployed"}}}`},
		{200, `{"data":{"type":"deployments","id":"d1","attributes":{"name":"svc","project_id":"p1","status":"deployed"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--template-values", `{"key":"val"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	raw, err := io.ReadAll(mt.calls[1].Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	if _, ok := attrs["template_values"]; !ok {
		t.Error("expected template_values in PATCH body")
	}
	// node_selector was not changed, must be absent
	if _, ok := attrs["node_selector"]; ok {
		t.Errorf("expected node_selector absent (not changed), but present: %v", attrs["node_selector"])
	}
}

// TestUpdate_MalformedStructuredInput verifies update returns error and no HTTP call on bad JSON.
func TestUpdate_MalformedStructuredInput(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{
		"--template-extra-resources", `not-valid`,
	}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{"d1"})
	if err == nil {
		t.Fatal("expected error for malformed template-extra-resources")
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected no HTTP calls on validation error, got %d", len(mt.calls))
	}
}

// TestUpdate_DryRunNoHTTP verifies update dry-run emits JSON and makes no HTTP calls.
func TestUpdate_DryRunNoHTTP(t *testing.T) {
	mt := &mockTransport{}
	ctx, out := buildCtx(t, mt, true)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{"--package-version", "3.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"d1"}); err != nil {
		t.Fatalf("update dry-run: %v", err)
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected no HTTP calls in dry-run, got %d", len(mt.calls))
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	attrs := mustAttrs(t, body)
	if attrs["package_version"] != "3.0.0" {
		t.Errorf("expected package_version=3.0.0, got %v", attrs["package_version"])
	}
	if _, ok := attrs["name"]; ok {
		t.Errorf("expected name absent (not changed) in dry-run, got: %v", attrs["name"])
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
