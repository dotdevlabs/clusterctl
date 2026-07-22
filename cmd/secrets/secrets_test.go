package secrets_test

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

	"github.com/dotdevlabs/clusterctl/cmd/secrets"
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
	cmd := secrets.NewCommand()
	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":[{"id":"s1","name":"mysecret","project_id":"p1"}]}`},
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
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"id":"s2","name":"newsecret","project_id":"p1"}}`},
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
	if err := sub.ParseFlags([]string{"--name", "newsecret", "--value", "myvalue"}); err != nil {
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
		{200, `{"status":"materialized"}`},
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
