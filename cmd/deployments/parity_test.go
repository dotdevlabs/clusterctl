package deployments_test

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/dotdevlabs/clusterctl/cmd/deployments"
)

// TestDeploymentRequestParity fails if the API spec's DeploymentRequest attributes
// contain a field that has no matching --flag-name in the create command.
func TestDeploymentRequestParity(t *testing.T) {
	data, err := os.ReadFile("../../docs/api_spec.yaml")
	if err != nil {
		t.Fatalf("reading api_spec.yaml: %v", err)
	}

	var spec map[string]any
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parsing api_spec.yaml: %v", err)
	}

	// Walk: components.schemas.DeploymentRequest.properties.data.properties.attributes.properties
	attrs := specWalk(t, spec,
		"components", "schemas", "DeploymentRequest",
		"properties", "data", "properties", "attributes", "properties",
	)

	createCmd := deployments.NewCommand()
	sub, _, err := createCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("finding create subcommand: %v", err)
	}

	for fieldName := range attrs {
		flagName := strings.ReplaceAll(fieldName, "_", "-")
		if sub.Flags().Lookup(flagName) == nil {
			t.Errorf("API spec field %q has no matching --%s flag in deployments create", fieldName, flagName)
		}
	}
}

// specWalk traverses nested map[string]any along the given path keys.
func specWalk(t *testing.T, node any, keys ...string) map[string]any {
	t.Helper()
	cur := node
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("spec path element %q: expected map, got %T", k, cur)
		}
		next, ok := m[k]
		if !ok {
			t.Fatalf("spec path element %q not found", k)
		}
		cur = next
	}
	result, ok := cur.(map[string]any)
	if !ok {
		t.Fatalf("spec leaf: expected map[string]any, got %T", cur)
	}
	return result
}
