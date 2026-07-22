package clusters_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/cmd/clusters"
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
	client := httpclient.NewWithTransport("https://example.com", "tok", transport)
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
		{200, `{"data":[{"id":"c1","name":"prod","cluster_type":"virtual","status":"ready"}]}`},
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
		{200, `{"data":{"id":"c1","name":"prod","cluster_type":"virtual","status":"ready"}}`},
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
		{201, `{"data":{"id":"c2","name":"dev","cluster_type":"virtual","status":"pending"}}`},
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
		{200, `{"data":{"id":"c1","name":"renamed","cluster_type":"virtual","status":"ready"}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := clusters.NewCommand()
	sub, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--name", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := sub.RunE(sub, []string{"c1"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Errorf("expected renamed in output, got: %s", out.String())
	}
	if mt.calls[0].Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", mt.calls[0].Method)
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
		{200, `{"status":"healthy"}`},
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
	if !strings.Contains(out.String(), "healthy") {
		t.Errorf("expected healthy in output, got: %s", out.String())
	}
}

func TestFluxBootstrap(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"status":"bootstrapped"}`},
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
