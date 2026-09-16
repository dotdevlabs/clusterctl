package packageupdatepolicies_test

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

	"github.com/dotdevlabs/clusterctl/cmd/packageupdatepolicies"
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

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"type":"package_update_policies","id":"p1","attributes":{"deployment_id":"d1","package_id":"pkg1","is_blocked":false}}],"links":{}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packageupdatepolicies.NewCommand()
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

func TestCreate_RequestBody(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"package_update_policies","id":"p2","attributes":{"deployment_id":"d1","package_id":"pkg1","is_blocked":false}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packageupdatepolicies.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--deployment-id", "d1", "--package-id", "pkg1"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if mt.calls[0].Method != http.MethodPost {
		t.Errorf("expected POST, got %s", mt.calls[0].Method)
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
	if data["type"] != "package_update_policies" {
		t.Errorf("expected data.type=package_update_policies, got %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["deployment_id"] != "d1" {
		t.Errorf("expected deployment_id=d1, got %v", attrs["deployment_id"])
	}
}

func TestCreate_ErrorResponse(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{422, `{"errors":[{"title":"Unprocessable Entity","detail":"deployment_id is required"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packageupdatepolicies.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--deployment-id", "d1"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{}); err == nil {
		t.Fatal("expected error for 422 response")
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"package_update_policies","id":"p1","attributes":{"deployment_id":"d1","package_id":"pkg1","is_blocked":false}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packageupdatepolicies.NewCommand()
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

func TestUpdate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"package_update_policies","id":"p1","links":{"self":"/api/v1/package_update_policies/p1"},"attributes":{"deployment_id":"d1","package_id":"pkg1","is_blocked":false}}}`},
		{200, `{"data":{"type":"package_update_policies","id":"p1","attributes":{"deployment_id":"d1","package_id":"pkg2","is_blocked":false}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := packageupdatepolicies.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--package-id", "pkg2"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "p1") {
		t.Errorf("expected p1 in output, got: %s", out.String())
	}
	if len(mt.calls) < 2 {
		t.Fatal("expected 2 HTTP calls (GET + PATCH)")
	}
	if mt.calls[1].Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", mt.calls[1].Method)
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
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	if attrs["package_id"] != "pkg2" {
		t.Errorf("expected package_id=pkg2, got %v", attrs["package_id"])
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"package_update_policies","id":"p1","links":{"self":"/api/v1/package_update_policies/p1"},"attributes":{"deployment_id":"d1","package_id":"pkg1","is_blocked":false}}}`},
		{204, ``},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := packageupdatepolicies.NewCommand()
	sub, _, err := parent.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.RunE(sub, []string{"p1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mt.calls) != 2 {
		t.Errorf("expected 2 HTTP calls (GET + DELETE), got %d", len(mt.calls))
	}
	if mt.calls[1].Method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", mt.calls[1].Method)
	}
}

func TestCreate_MaxAttempts(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"package_update_policies","id":"p3","attributes":{"deployment_id":"d1","max_attempts":5}}}`},
	}}
	ctx, _ := buildCtx(t, mt, true)
	parent := packageupdatepolicies.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--deployment-id", "d1", "--max-attempts", "5"}); err != nil {
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
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be an object, got: %T", data["attributes"])
	}
	maxAttempts, ok := attrs["max_attempts"].(float64)
	if !ok {
		t.Fatalf("expected max_attempts to be a number, got: %T (%v)", attrs["max_attempts"], attrs["max_attempts"])
	}
	if int(maxAttempts) != 5 {
		t.Errorf("expected max_attempts=5, got %v", maxAttempts)
	}
}
