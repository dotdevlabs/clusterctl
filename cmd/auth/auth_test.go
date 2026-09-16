package auth_test

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

	"github.com/dotdevlabs/clusterctl/cmd/auth"
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

func TestWhoami(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{200, `{"data":{"type":"auth_contexts","id":"1","attributes":{"organization":{"id":"org1","name":"Acme Corp","slug":"acme"},"owner":{"id":"u1","email_address":"admin@acme.com"},"token":{"id":"tok1","name":"mytoken"}}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := auth.NewCommand()
	sub, _, err := parent.Find([]string{"whoami"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("whoami: %v", err)
	}
	if len(mt.calls) == 0 {
		t.Fatal("expected HTTP call")
	}
	if mt.calls[0].Method != http.MethodGet {
		t.Errorf("expected GET, got %s", mt.calls[0].Method)
	}
	if mt.calls[0].URL.Path != "/api/v1/auth" {
		t.Errorf("expected path /api/v1/auth, got %s", mt.calls[0].URL.Path)
	}
	if !strings.Contains(out.String(), "Acme Corp") {
		t.Errorf("expected organization name in output, got: %s", out.String())
	}
}

func TestWhoami_ErrorResponse(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{401, `{"errors":[{"status":"401","title":"Unauthorized"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := auth.NewCommand()
	sub, _, err := parent.Find([]string{"whoami"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
