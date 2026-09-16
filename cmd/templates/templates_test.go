package templates_test

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

	"github.com/dotdevlabs/clusterctl/cmd/templates"
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
