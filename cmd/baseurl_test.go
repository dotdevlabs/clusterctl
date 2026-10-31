package cmd

import (
	"bytes"
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
