package cmd

import (
	"io"
	"strings"

	"bytes"
	"github.com/dotdevlabs/ctlkit/pkg/root/roottest"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBaseURLEnvOverride runs a real command with no context file and only
// CLUSTERCTL_BASE_URL set; the request must reach the stub HTTP server.
func TestBaseURLEnvOverride(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CLUSTERCTL_CONTEXT", "")
	t.Setenv("CLUSTERCTL_BASE_URL", srv.URL)
	t.Setenv("CLUSTERCTL_TOKEN", "env-token")

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"projects", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, out.String())
	}
	if gotPath == "" {
		t.Fatal("request did not reach the CLUSTERCTL_BASE_URL server")
	}
	if gotAuth != "Bearer env-token" {
		t.Errorf("Authorization = %q, want bearer env-token", gotAuth)
	}
}

// stubTransport records requests and answers with an empty JSON:API list.
type stubTransport struct {
	reqs []*http.Request
}

func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.reqs = append(s.reqs, r)
	body := `{"data":[]}`
	if strings.HasSuffix(r.URL.Path, "/auth") {
		body = `{"data":{"id":"1","type":"auth_contexts","attributes":{}}}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/vnd.api+json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

// withStubTransport swaps http.DefaultTransport for the test's duration.
func withStubTransport(t *testing.T) *stubTransport {
	t.Helper()
	stub := &stubTransport{}
	orig := http.DefaultTransport
	http.DefaultTransport = stub
	t.Cleanup(func() { http.DefaultTransport = orig })
	return stub
}

func isolateConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CLUSTERCTL_CONTEXT", "")
	t.Setenv("CLUSTERCTL_BASE_URL", "")
	t.Setenv("CLUSTERCTL_TOKEN", "")
}

func runRoot(t *testing.T, args ...string) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, out.String())
	}
}

func TestRootBuildsWithDefaultBaseURL(t *testing.T) {
	roottest.AssertBuilds(t, buildConfig())
	if got := buildConfig().DefaultBaseURL; got != "https://clustercontrol.co" {
		t.Errorf("DefaultBaseURL = %q", got)
	}
}

// TestDefaultBaseURLTargeted: no context file and only the token env var must
// target https://clustercontrol.co, for a normal command and for auth whoami.
func TestDefaultBaseURLTargeted(t *testing.T) {
	for _, args := range [][]string{
		{"projects", "list", "--json"},
		{"auth", "whoami", "--json"},
	} {
		t.Run(args[0]+"_"+args[1], func(t *testing.T) {
			isolateConfig(t)
			t.Setenv("CLUSTERCTL_TOKEN", "env-token")
			stub := withStubTransport(t)
			runRoot(t, args...)
			if len(stub.reqs) == 0 {
				t.Fatal("no request issued")
			}
			r := stub.reqs[0]
			if r.URL.Scheme != "https" || r.URL.Host != "clustercontrol.co" {
				t.Errorf("request went to %s://%s", r.URL.Scheme, r.URL.Host)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer env-token" {
				t.Errorf("Authorization = %q", got)
			}
		})
	}
}

func TestWhoamiBaseURLOverrideAndTokenFlag(t *testing.T) {
	isolateConfig(t)
	t.Setenv("CLUSTERCTL_BASE_URL", "https://staging.example.com")
	t.Setenv("CLUSTERCTL_TOKEN", "env-token")
	stub := withStubTransport(t)
	runRoot(t, "auth", "whoami", "--json", "--token", "flag-token")
	if len(stub.reqs) == 0 {
		t.Fatal("no request issued")
	}
	r := stub.reqs[0]
	if r.URL.Host != "staging.example.com" {
		t.Errorf("host = %q, want override", r.URL.Host)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer flag-token" {
		t.Errorf("Authorization = %q, want flag token", got)
	}
}
