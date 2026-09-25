package conformance_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/cmd/auth"
	"github.com/dotdevlabs/clusterctl/cmd/clusters"
	"github.com/dotdevlabs/clusterctl/cmd/deployments"
	"github.com/dotdevlabs/clusterctl/cmd/packages"
	"github.com/dotdevlabs/clusterctl/cmd/packageupdatepolicies"
	"github.com/dotdevlabs/clusterctl/cmd/projects"
	"github.com/dotdevlabs/clusterctl/cmd/registrations"
	"github.com/dotdevlabs/clusterctl/cmd/secrets"
	"github.com/dotdevlabs/clusterctl/cmd/status"
	"github.com/dotdevlabs/clusterctl/cmd/templates"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

func buildRoot() *cobra.Command {
	root := &cobra.Command{Use: "clusterctl"}
	root.AddCommand(
		auth.NewCommand(),
		clusters.NewCommand(),
		deployments.NewCommand(),
		packages.NewCommand(),
		packageupdatepolicies.NewCommand(),
		projects.NewCommand(),
		registrations.NewCommand(),
		secrets.NewCommand(),
		status.NewCommand(),
		templates.NewCommand(),
	)
	return root
}

// opCommandMap maps each spec operationId to its cobra command path in the root.
var opCommandMap = map[string][]string{
	"getStatus":                      {"status"},
	"getAuth":                        {"auth", "whoami"},
	"createRegistration":             {"registrations", "create"},
	"listClusters":                   {"clusters", "list"},
	"createCluster":                  {"clusters", "create"},
	"getCluster":                     {"clusters", "get"},
	"updateCluster":                  {"clusters", "update"},
	"deleteCluster":                  {"clusters", "delete"},
	"createClusterHealthCheck":       {"clusters", "health-check"},
	"getClusterProvisioning":         {"clusters", "provisioning"},
	"createClusterProvisioning":      {"clusters", "retry-provisioning"},
	"getClusterFluxBootstrap":        {"clusters", "flux-bootstrap-status"},
	"createClusterFluxBootstrap":     {"clusters", "flux-bootstrap"},
	"createClusterExposure":          {"clusters", "expose"},
	"listProjects":                   {"projects", "list"},
	"createProject":                  {"projects", "create"},
	"getProject":                     {"projects", "get"},
	"updateProject":                  {"projects", "update"},
	"deleteProject":                  {"projects", "delete"},
	"listProjectSecrets":             {"secrets", "list"},
	"createProjectSecret":            {"secrets", "create"},
	"deleteProjectSecret":            {"secrets", "delete"},
	"createSecretMaterialization":    {"secrets", "materialize"},
	"listPackages":                   {"packages", "list"},
	"createPackage":                  {"packages", "create"},
	"getPackage":                     {"packages", "get"},
	"updatePackage":                  {"packages", "update"},
	"deletePackage":                  {"packages", "delete"},
	"listPackageReleases":            {"packages", "releases", "list"},
	"getPackageRelease":              {"packages", "releases", "get"},
	"listPackageUpdatePolicies":      {"package-update-policies", "list"},
	"createPackageUpdatePolicy":      {"package-update-policies", "create"},
	"getPackageUpdatePolicy":         {"package-update-policies", "get"},
	"updatePackageUpdatePolicy":      {"package-update-policies", "update"},
	"deletePackageUpdatePolicy":      {"package-update-policies", "delete"},
	"listDeployments":                {"deployments", "list"},
	"createDeployment":               {"deployments", "create"},
	"getDeployment":                  {"deployments", "get"},
	"updateDeployment":               {"deployments", "update"},
	"deleteDeployment":               {"deployments", "delete"},
	"listDeploymentUpdateRuns":       {"deployments", "update-runs", "list"},
	"getDeploymentUpdateRun":         {"deployments", "update-runs", "get"},
	"pinDeployment":                  {"deployments", "pin"},
	"unpinDeployment":                {"deployments", "unpin"},
	"previewDeploymentPackageUpdate": {"deployments", "package-update", "get"},
	"triggerDeploymentPackageUpdate": {"deployments", "package-update", "apply"},
	"rolloutDeployment":              {"deployments", "rollout"},
	"listDeploymentImageRevisions":   {"deployments", "revisions"},
	"rollbackDeployment":             {"deployments", "rollback"},
	"releaseDeploymentImageHold":     {"deployments", "rollback", "release"},
	"listTemplates":                  {"templates", "list"},
	"getTemplate":                    {"templates", "get"},
	"createTemplate":                 {"templates", "create"},
	"updateTemplate":                 {"templates", "update"},
}

// connectorOps are intentionally excluded from CLI coverage (in-cluster connector, not operators).
var connectorOps = map[string]bool{
	"connectorHeartbeatLegacy": true,
	"connectorHeartbeat":       true,
	"connectorInventory":       true,
	"connectorFluxEvents":      true,
	"connectorFlaggerEvents":   true,
}

func TestOperationCoverage(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(filename), "..", "..", "docs", "api_spec.yaml")

	data, err := os.ReadFile(specPath) //nolint:gosec
	if err != nil {
		t.Fatalf("reading api_spec.yaml: %v", err)
	}

	var spec map[string]any
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parsing api_spec.yaml: %v", err)
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("spec.paths is not a map")
	}

	root := buildRoot()

	for _, pathItem := range paths {
		item, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			op, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			opID, ok := op["operationId"].(string)
			if !ok {
				continue
			}

			if connectorOps[opID] {
				t.Logf("skipping connector op: %s", opID)
				continue
			}

			cmdPath, ok := opCommandMap[opID]
			if !ok {
				t.Errorf("operationId %q is in the spec but not in opCommandMap — add it to the map and implement the command", opID)
				continue
			}

			t.Logf("checking %s -> %v", opID, cmdPath)

			found, _, err := root.Find(cmdPath)
			if err != nil || found == nil || found == root {
				t.Errorf("operationId %q maps to command path %v but that command was not found in the cobra tree", opID, cmdPath)
			}
		}
	}
}

// TestConnectorOpsExclusionIsExplicit verifies that connectorOps is non-empty so the
// test cannot pass by accident if the exclusion list is accidentally emptied.
func TestConnectorOpsExclusionIsExplicit(t *testing.T) {
	if len(connectorOps) == 0 {
		t.Fatal("connectorOps must be non-empty — connector endpoints are intentionally excluded")
	}
}

func extractAttrs(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data to be object, got %T", body["data"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected body.data.attributes to be object, got %T", data["attributes"])
	}
	return attrs
}

// conformanceMockTransport is a minimal RoundTripper for conformance assertions.
type conformanceMockTransport struct {
	responses []conformanceMockResponse
	calls     []*http.Request
}

type conformanceMockResponse struct {
	status int
	body   string
}

func (m *conformanceMockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, r)
	if len(m.responses) == 0 {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	}
	resp := m.responses[0]
	m.responses = m.responses[1:]
	return &http.Response{StatusCode: resp.status, Body: io.NopCloser(strings.NewReader(resp.body)), Header: make(http.Header)}, nil
}

// TestPackageRenderModeConformance pins the HTTP wire format for render_mode on package create.
func TestPackageRenderModeConformance(t *testing.T) {
	packageResp := `{"data":{"type":"packages","id":"pkg1","attributes":{"name":"mypkg","source_type":"helm"}}}`

	t.Run("set render_mode slug", func(t *testing.T) {
		mt := &conformanceMockTransport{responses: []conformanceMockResponse{
			{201, packageResp},
		}}
		var out, errOut strings.Builder
		client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: mt})
		renderer := output.New(false, "", &out, &errOut)
		ctx := context.Background()
		ctx = ctxutil.WithClient(ctx, client)
		ctx = ctxutil.WithRenderer(ctx, renderer)
		ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{})

		root := buildRoot()
		sub, _, err := root.Find([]string{"packages", "create"})
		if err != nil || sub == nil {
			t.Fatal("could not find 'packages create' command")
		}
		sub.SetContext(ctx)
		if err := sub.ParseFlags([]string{"--name", "mypkg", "--render-mode", "slug"}); err != nil {
			t.Fatal(err)
		}
		if err := sub.RunE(sub, []string{}); err != nil {
			t.Fatalf("packages create: %v", err)
		}
		if len(mt.calls) != 1 {
			t.Fatalf("expected 1 HTTP call, got %d", len(mt.calls))
		}
		raw, err := io.ReadAll(mt.calls[0].Body)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		attrs := extractAttrs(t, body)
		if attrs["render_mode"] != "slug" {
			t.Errorf("expected render_mode=slug, got %v", attrs["render_mode"])
		}
	})

	t.Run("clear render_mode sends null", func(t *testing.T) {
		mt := &conformanceMockTransport{responses: []conformanceMockResponse{
			{201, packageResp},
		}}
		var out, errOut strings.Builder
		client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: mt})
		renderer := output.New(false, "", &out, &errOut)
		ctx := context.Background()
		ctx = ctxutil.WithClient(ctx, client)
		ctx = ctxutil.WithRenderer(ctx, renderer)
		ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{})

		root := buildRoot()
		sub, _, err := root.Find([]string{"packages", "create"})
		if err != nil || sub == nil {
			t.Fatal("could not find 'packages create' command")
		}
		sub.SetContext(ctx)
		if err := sub.ParseFlags([]string{"--name", "mypkg", "--clear-render-mode"}); err != nil {
			t.Fatal(err)
		}
		if err := sub.RunE(sub, []string{}); err != nil {
			t.Fatalf("packages create: %v", err)
		}
		if len(mt.calls) != 1 {
			t.Fatalf("expected 1 HTTP call, got %d", len(mt.calls))
		}
		raw, err := io.ReadAll(mt.calls[0].Body)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		attrs := extractAttrs(t, body)
		val, present := attrs["render_mode"]
		if !present {
			t.Error("expected render_mode key present (null), but key was absent")
		}
		if val != nil {
			t.Errorf("expected render_mode=null, got %v", val)
		}
	})

	t.Run("no render_mode flag omits key", func(t *testing.T) {
		mt := &conformanceMockTransport{responses: []conformanceMockResponse{
			{201, packageResp},
		}}
		var out, errOut strings.Builder
		client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: mt})
		renderer := output.New(false, "", &out, &errOut)
		ctx := context.Background()
		ctx = ctxutil.WithClient(ctx, client)
		ctx = ctxutil.WithRenderer(ctx, renderer)
		ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{})

		root := buildRoot()
		sub, _, err := root.Find([]string{"packages", "create"})
		if err != nil || sub == nil {
			t.Fatal("could not find 'packages create' command")
		}
		sub.SetContext(ctx)
		if err := sub.ParseFlags([]string{"--name", "mypkg"}); err != nil {
			t.Fatal(err)
		}
		if err := sub.RunE(sub, []string{}); err != nil {
			t.Fatalf("packages create: %v", err)
		}
		if len(mt.calls) != 1 {
			t.Fatalf("expected 1 HTTP call, got %d", len(mt.calls))
		}
		raw, err := io.ReadAll(mt.calls[0].Body)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		attrs := extractAttrs(t, body)
		if _, present := attrs["render_mode"]; present {
			t.Errorf("expected render_mode key absent, but it was present: %v", attrs["render_mode"])
		}
	})
}

// TestSecretsDeleteConformance asserts that "secrets delete" sends exactly one DELETE
// request and never calls the spec-absent GET /secrets/{id} endpoint.
// The ClusterControl API publishes only index, create, and destroy for project secrets;
// any GET call before DELETE would violate the published spec.
func TestSecretsDeleteConformance(t *testing.T) {
	mt := &conformanceMockTransport{responses: []conformanceMockResponse{
		{204, ``},
	}}

	var out, errOut strings.Builder
	client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: mt})
	renderer := output.New(false, "", &out, &errOut)
	ctx := context.Background()
	ctx = ctxutil.WithClient(ctx, client)
	ctx = ctxutil.WithRenderer(ctx, renderer)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{})

	root := buildRoot()
	if err := root.PersistentFlags().Set("project-id", "p1"); err != nil {
		// project-id is on the secrets subcommand, not root; set it below
		_ = err
	}

	secretsCmd, _, err := root.Find([]string{"secrets"})
	if err != nil || secretsCmd == nil {
		t.Fatal("could not find 'secrets' command")
	}
	if err := secretsCmd.PersistentFlags().Set("project-id", "p1"); err != nil {
		t.Fatalf("set --project-id: %v", err)
	}

	deleteCmd, _, err := root.Find([]string{"secrets", "delete"})
	if err != nil || deleteCmd == nil {
		t.Fatal("could not find 'secrets delete' command")
	}
	deleteCmd.SetContext(ctx)

	if err := deleteCmd.RunE(deleteCmd, []string{"s1"}); err != nil {
		t.Fatalf("secrets delete: %v", err)
	}

	if len(mt.calls) != 1 {
		t.Errorf("expected exactly 1 HTTP call, got %d — a GET before DELETE would violate the published spec", len(mt.calls))
	}
	if len(mt.calls) > 0 && mt.calls[0].Method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", mt.calls[0].Method)
	}
	if len(mt.calls) > 0 && !strings.Contains(mt.calls[0].URL.Path, "/secrets/s1") {
		t.Errorf("expected /secrets/s1 in DELETE path, got: %s", mt.calls[0].URL.Path)
	}
}
