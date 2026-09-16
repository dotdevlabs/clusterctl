package registrations_test

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

	"github.com/dotdevlabs/clusterctl/cmd/registrations"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

type mockTransport struct {
	responses []mockResponse
	calls     []*http.Request
	bodies    [][]byte
}

type mockResponse struct {
	status int
	body   string
}

func (m *mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, r)
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		m.bodies = append(m.bodies, raw)
	} else {
		m.bodies = append(m.bodies, nil)
	}
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

func TestCreate_FlatBody(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{201, `{"data":{"type":"registrations","id":"reg1","attributes":{"token":"tok-abc","organization_id":"org1","owner_id":"u1"}}}`},
	}}
	ctx, out := buildCtx(t, mt, true)
	parent := registrations.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)
	if err := sub.ParseFlags([]string{"--owner-email", "admin@acme.com", "--label", "mytoken"}); err != nil {
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
	if mt.calls[0].URL.Path != "/api/v1/registrations" {
		t.Errorf("expected path /api/v1/registrations, got %s", mt.calls[0].URL.Path)
	}
	raw := mt.bodies[0]
	if raw == nil {
		t.Fatal("expected request body")
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if _, hasData := body["data"]; hasData {
		t.Error("expected flat body without JSON:API envelope (no 'data' key)")
	}
	if body["owner_email"] != "admin@acme.com" {
		t.Errorf("expected owner_email=admin@acme.com, got %v", body["owner_email"])
	}
	if body["label"] != "mytoken" {
		t.Errorf("expected label=mytoken, got %v", body["label"])
	}
	if !strings.Contains(out.String(), "tok-abc") {
		t.Errorf("expected token in output, got: %s", out.String())
	}
}

func TestCreate_ErrorResponse(t *testing.T) {
	mt := &mockTransport{responses: []mockResponse{
		{422, `{"errors":[{"status":"422","title":"Unprocessable Entity","detail":"owner_email is invalid"}]}`},
	}}
	ctx, _ := buildCtx(t, mt, false)
	parent := registrations.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	sub.SetContext(ctx)
	if err := sub.ParseFlags([]string{"--owner-email", "bad", "--label", "test"}); err != nil {
		t.Fatal(err)
	}
	err = sub.RunE(sub, []string{})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}
