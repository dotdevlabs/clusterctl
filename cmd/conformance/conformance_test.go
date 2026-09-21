package conformance_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

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
