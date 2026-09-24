package templates_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/cmd/templates"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

type mockTransport struct {
	responses []mockResponse
	calls     []*http.Request
	bodies    []string
}

type mockResponse struct {
	status int
	body   string
}

func (m *mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		m.bodies = append(m.bodies, string(b))
		_ = r.Body.Close()
	} else {
		m.bodies = append(m.bodies, "")
	}
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

func buildCtxDryRun(t *testing.T, transport http.RoundTripper, jsonMode bool) (context.Context, *bytes.Buffer) {
	t.Helper()
	var out, errOut bytes.Buffer
	client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: transport})
	renderer := output.New(jsonMode, "", &out, &errOut)
	ctx := context.Background()
	ctx = ctxutil.WithClient(ctx, client)
	ctx = ctxutil.WithRenderer(ctx, renderer)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: jsonMode, DryRun: true})
	return ctx, &out
}

// ── Existing command regression ───────────────────────────────────────────────

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"templates","id":"t1","attributes":{"slug":"basic","name":"Basic Template"}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if mt.calls[0].Method != http.MethodGet {
		t.Errorf("expected GET, got %s", mt.calls[0].Method)
	}
	if !strings.Contains(out.String(), "t1") {
		t.Errorf("expected t1 in output, got: %s", out.String())
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"basic","name":"Basic Template","description":"A basic template"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if mt.calls[0].Method != http.MethodGet {
		t.Errorf("expected GET, got %s", mt.calls[0].Method)
	}
	if !strings.Contains(out.String(), "t1") {
		t.Errorf("expected t1 in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "basic") {
		t.Errorf("expected slug 'basic' in output, got: %s", out.String())
	}
}

func TestGet_ErrorResponse(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{404, `{"errors":[{"status":"404","title":"Not Found"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	err = sub.RunE(sub, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestList_FullAttributes(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n","manifest_files":{"deploy.yaml":"apiVersion: v1"},"inputs":[{"key":"img","type":"string","required":true}]}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "manifest_files") {
		t.Errorf("expected manifest_files in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "inputs") {
		t.Errorf("expected inputs in output, got: %s", out.String())
	}
}

func TestGet_FullAttributes(t *testing.T) {
	multilineYAML := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test"
	body := `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n","manifest_files":{"cm.yaml":"` + strings.ReplaceAll(multilineYAML, "\n", "\\n") + `"},"inputs":[{"key":"img","type":"string","default":null,"required":true,"description":null}]}}}`
	mt := &mockTransport{responses: []mockResponse{
		{200, body},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "manifest_files") {
		t.Errorf("expected manifest_files in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "inputs") {
		t.Errorf("expected inputs in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "cm.yaml") {
		t.Errorf("expected cm.yaml key in output, got: %s", out.String())
	}
}

// ── Create command ────────────────────────────────────────────────────────────

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"templates","id":"t99","attributes":{"slug":"s","name":"n"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("slug", "s"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", len(mt.calls))
	}
	if mt.calls[0].Method != http.MethodPost {
		t.Errorf("expected POST, got %s", mt.calls[0].Method)
	}
	if !strings.Contains(mt.calls[0].URL.Path, "/templates") {
		t.Errorf("expected /templates in path, got: %s", mt.calls[0].URL.Path)
	}
	if !strings.Contains(mt.bodies[0], `"type":"templates"`) {
		t.Errorf("expected type:templates in body, got: %s", mt.bodies[0])
	}
	if !strings.Contains(mt.bodies[0], `"slug":"s"`) {
		t.Errorf("expected slug in body, got: %s", mt.bodies[0])
	}
	if !strings.Contains(mt.bodies[0], `"name":"n"`) {
		t.Errorf("expected name in body, got: %s", mt.bodies[0])
	}
	if !strings.Contains(out.String(), "t99") {
		t.Errorf("expected id t99 in output, got: %s", out.String())
	}
}

func TestCreate_DryRun(t *testing.T) {
	mt := &mockTransport{}
	ctx, out := buildCtxDryRun(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.Flags().Set("slug", "s"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls for dry-run, got %d", len(mt.calls))
	}
	if !strings.Contains(out.String(), `"type"`) || !strings.Contains(out.String(), `"templates"`) {
		t.Errorf("expected request body in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `"slug"`) {
		t.Errorf("expected slug in request body, got: %s", out.String())
	}
}

func TestCreate_InvalidInputsJSON(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("slug", "s"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("inputs", "bad json"); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for invalid inputs JSON")
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls, got %d", len(mt.calls))
	}
}

func TestCreate_InvalidManifestFilesJSON(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("slug", "s"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("manifest-files", "not-json"); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for invalid manifest-files JSON")
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls, got %d", len(mt.calls))
	}
}

func TestCreate_422Error(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{422, `{"errors":[{"status":"422","title":"Unprocessable Entity","detail":"slug is invalid"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("slug", "s"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}

func TestCreate_FileInput(t *testing.T) {
	inputData := `[{"key":"image","type":"string","required":true}]`
	f, err := os.CreateTemp(t.TempDir(), "inputs*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(inputData); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n"}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("slug", "s"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	if err := sub.Flags().Set("inputs", "@"+f.Name()); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create with @file: %v", err)
	}
	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", len(mt.calls))
	}
	if !strings.Contains(mt.bodies[0], `"image"`) {
		t.Errorf("expected file content in body, got: %s", mt.bodies[0])
	}
}

func TestCreate_RequiredFlags(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	// only set name, not slug — cobra should fail before RunE
	if err := sub.Flags().Set("name", "n"); err != nil {
		t.Fatal(err)
	}
	err = sub.ValidateRequiredFlags()
	if err == nil {
		t.Fatal("expected error for missing required --slug")
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls, got %d", len(mt.calls))
	}
}

// ── Update command ────────────────────────────────────────────────────────────

func TestUpdate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n","description":"updated"},"meta":{"rerendered_deployments":["d1"],"rerender_errors":[]}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("description", "updated"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(mt.calls) != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", len(mt.calls))
	}
	if mt.calls[0].Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", mt.calls[0].Method)
	}
	if !strings.Contains(mt.calls[0].URL.Path, "/templates/t1") {
		t.Errorf("expected /templates/t1 in path, got: %s", mt.calls[0].URL.Path)
	}
	if !strings.Contains(out.String(), "t1") {
		t.Errorf("expected t1 in output, got: %s", out.String())
	}
}

func TestUpdate_DryRun(t *testing.T) {
	mt := &mockTransport{}
	ctx, out := buildCtxDryRun(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.Flags().Set("description", "updated"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update dry-run: %v", err)
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls for dry-run, got %d", len(mt.calls))
	}
	if !strings.Contains(out.String(), `"type"`) || !strings.Contains(out.String(), `"templates"`) {
		t.Errorf("expected request body in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `"description"`) {
		t.Errorf("expected description in request body, got: %s", out.String())
	}
}

func TestUpdate_OmitUnset(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n","description":"desc"},"meta":{"rerendered_deployments":[],"rerender_errors":[]}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("description", "desc"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if strings.Contains(mt.bodies[0], "inputs") {
		t.Errorf("inputs should be absent when not provided, got: %s", mt.bodies[0])
	}
	if strings.Contains(mt.bodies[0], "manifest_files") {
		t.Errorf("manifest_files should be absent when not provided, got: %s", mt.bodies[0])
	}
}

func TestUpdate_ExplicitEmptyInputs(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n"},"meta":{"rerendered_deployments":[],"rerender_errors":[]}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("inputs", "[]"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(mt.bodies[0], `"inputs":[]`) {
		t.Errorf("expected inputs:[] in body, got: %s", mt.bodies[0])
	}
}

func TestUpdate_ExplicitEmptyManifestFiles(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n"},"meta":{"rerendered_deployments":[],"rerender_errors":[]}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("manifest-files", "{}"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(mt.bodies[0], `"manifest_files":{}`) {
		t.Errorf("expected manifest_files:{} in body, got: %s", mt.bodies[0])
	}
}

func TestUpdate_ClearDescription(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n"},"meta":{"rerendered_deployments":[],"rerender_errors":[]}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("clear-description", "true"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(mt.bodies[0], `"description":null`) {
		t.Errorf("expected description:null in body, got: %s", mt.bodies[0])
	}
}

func TestUpdate_NoFlags(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	err = sub.RunE(sub, []string{"t1"})
	if err == nil {
		t.Fatal("expected error when no flags provided")
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls, got %d", len(mt.calls))
	}
}

func TestUpdate_RerenderErrors(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n"},"meta":{"rerendered_deployments":["d1"],"rerender_errors":["render failed for d2"]}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("description", "x"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "rerender_errors") {
		t.Errorf("expected rerender_errors in JSON output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "render failed for d2") {
		t.Errorf("expected error message in JSON output, got: %s", out.String())
	}
}

func TestUpdate_404Error(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{404, `{"errors":[{"status":"404","title":"Not Found"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("description", "x"); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{"t1"})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestUpdate_InvalidInputsJSON(t *testing.T) {
	mt := &mockTransport{}
	ctx, _ := buildCtx(t, mt, false)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("inputs", "bad json"); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{"t1"})
	if err == nil {
		t.Fatal("expected error for invalid inputs JSON")
	}
	if len(mt.calls) != 0 {
		t.Errorf("expected 0 HTTP calls, got %d", len(mt.calls))
	}
}

func TestUpdate_JSONOutput_IncludesMetadata(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"templates","id":"t1","attributes":{"slug":"s","name":"n"},"meta":{"rerendered_deployments":["d1","d2"],"rerender_errors":["err1"]}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := templates.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.Flags().Set("inputs", "[]"); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"t1"}); err != nil {
		t.Fatalf("update: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, out.String())
	}
	if _, ok := result["rerendered_deployments"]; !ok {
		t.Errorf("expected rerendered_deployments in JSON output, got: %s", out.String())
	}
	if _, ok := result["rerender_errors"]; !ok {
		t.Errorf("expected rerender_errors in JSON output, got: %s", out.String())
	}
}
