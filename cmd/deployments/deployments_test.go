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
	client := httpclient.NewWithTransport("https://example.com", "tok", transport)
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
		{200, `{"data":[{"id":"d1","name":"my-deploy","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"deployed"}]}`},
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
		{200, `{"data":{"id":"d1","name":"my-deploy","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"deployed"}}`},
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
		{201, `{"data":{"id":"d2","name":"my-deploy","project_id":"p1","cluster_id":"c1","namespace":"default","package_name":"promtail","status":"pending"}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--project-id", "p1", "--cluster-id", "c1", "--name", "my-deploy", "--namespace", "default", "--package-name", "promtail"}); err != nil {
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
		{201, `{"data":{"id":"d3","name":"host-log-collector","namespace":"default","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"pending"}}`},
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
	if body["name"] != "host-log-collector" {
		t.Errorf("expected name=host-log-collector, got %v", body["name"])
	}
	if body["namespace"] != "default" {
		t.Errorf("expected namespace=default, got %v", body["namespace"])
	}
	if body["package_name"] != "promtail" {
		t.Errorf("expected package_name=promtail, got %v", body["package_name"])
	}
	if _, ok := body["package_id"]; ok {
		t.Errorf("expected package_id to be absent, but it was present: %v", body["package_id"])
	}
}

func TestUpdate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"id":"d1","name":"my-deploy","project_id":"p1","cluster_id":"c1","package_name":"promtail","status":"deployed"}}`},
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
		{200, `{"data":{"id":"d1","name":"my-deploy","project_id":"p1","cluster_id":"c1","namespace":"kube-system","package_name":"promtail-new","status":"deployed"}}`},
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
	if body["package_name"] != "promtail-new" {
		t.Errorf("expected package_name=promtail-new, got %v", body["package_name"])
	}
	if body["namespace"] != "kube-system" {
		t.Errorf("expected namespace=kube-system, got %v", body["namespace"])
	}
	if _, ok := body["package_id"]; ok {
		t.Errorf("expected package_id to be absent, but it was present: %v", body["package_id"])
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
