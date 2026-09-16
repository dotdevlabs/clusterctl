package status_test

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

	"github.com/dotdevlabs/clusterctl/cmd/status"
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

func TestStatus(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"status","id":"1","attributes":{"version":"1.2.3","sha":"abc123","db_version":"20240101000000"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	cmd := status.NewCommand()
	cmd.SetContext(ctx)
	cmd.SetOut(out)
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if mt.calls[0].Method != http.MethodGet {
		t.Errorf("expected GET, got %s", mt.calls[0].Method)
	}
	if mt.calls[0].URL.Path != "/api/v1/status" {
		t.Errorf("expected path /api/v1/status, got %s", mt.calls[0].URL.Path)
	}
	if !strings.Contains(out.String(), "20240101000000") {
		t.Errorf("expected db_version in output, got: %s", out.String())
	}
}

func TestStatus_ErrorResponse(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{404, `{"errors":[{"status":"404","title":"Not Found"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	cmd := status.NewCommand()
	cmd.SetContext(ctx)
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
