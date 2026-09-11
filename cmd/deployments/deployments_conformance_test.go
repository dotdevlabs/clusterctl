package deployments_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/cmd/deployments"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

// specFieldToFlag converts a snake_case spec field name to a kebab-case CLI flag name.
func specFieldToFlag(field string) string {
	return strings.ReplaceAll(field, "_", "-")
}

// loadWritableSpecFields parses docs/api_spec.yaml and returns all property keys
// under DeploymentRequest.data.attributes.properties.
func loadWritableSpecFields(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("../../docs/api_spec.yaml")
	if err != nil {
		t.Fatalf("loadWritableSpecFields: read api_spec.yaml: %v", err)
	}

	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("loadWritableSpecFields: parse yaml: %v", err)
	}

	nav := func(m map[string]interface{}, keys ...string) (map[string]interface{}, bool) {
		cur := m
		for _, k := range keys {
			next, ok := cur[k].(map[string]interface{})
			if !ok {
				return nil, false
			}
			cur = next
		}
		return cur, true
	}

	components, ok := nav(root, "components", "schemas", "DeploymentRequest", "properties", "data", "properties", "attributes", "properties")
	if !ok {
		t.Fatal("loadWritableSpecFields: could not navigate to DeploymentRequest.data.attributes.properties")
	}

	fields := make([]string, 0, len(components))
	for k := range components {
		fields = append(fields, k)
	}
	return fields
}

// buildConformanceCtx creates a context suitable for conformance dry-run tests.
func buildConformanceCtx(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()
	var out, errOut bytes.Buffer
	client := httpclient.NewWithTransport("https://example.com", "tok", &jsonapi.Transport{Wrapped: &mockTransport{}})
	renderer := output.New(true, "", &out, &errOut)
	ctx := context.Background()
	ctx = ctxutil.WithClient(ctx, client)
	ctx = ctxutil.WithRenderer(ctx, renderer)
	ctx = ctxutil.WithGlobalFlags(ctx, ctxutil.GlobalFlags{JSON: true, DryRun: true})
	return ctx, &out
}

// TestDeploymentContractConformance_FlagPresence verifies every writable spec field
// has a matching CLI flag on both create and update commands.
func TestDeploymentContractConformance_FlagPresence(t *testing.T) {
	fields := loadWritableSpecFields(t)
	parent := deployments.NewCommand()

	createCmd, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatalf("find create: %v", err)
	}
	updateCmd, _, err := parent.Find([]string{"update"})
	if err != nil {
		t.Fatalf("find update: %v", err)
	}

	for _, field := range fields {
		flagName := specFieldToFlag(field)
		if createCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("create: missing flag --%s for spec field %s", flagName, field)
		}
		if updateCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("update: missing flag --%s for spec field %s", flagName, field)
		}
	}
}

// TestDeploymentContractConformance_CreateDryRunBody verifies that a dry-run create
// with all spec fields set produces data.attributes containing every spec field.
func TestDeploymentContractConformance_CreateDryRunBody(t *testing.T) {
	fields := loadWritableSpecFields(t)

	ctx, out := buildConformanceCtx(t)
	parent := deployments.NewCommand()
	sub, _, err := parent.Find([]string{"create"})
	if err != nil {
		t.Fatalf("find create: %v", err)
	}
	sub.SetContext(ctx)
	sub.SetOut(out)

	args := []string{
		"--project-id", "p1",
		"--name", "conformance-test",
		"--namespace", "ns1",
		"--package-name", "pkg",
		"--package-version", "1.0.0",
		"--cluster-id", "c1",
		"--environment-preset", "dev",
		"--values-override", "key: val",
		"--is-ai",
		"--scaling-mode", "hpa",
		"--min-replicas", "1",
		"--max-replicas", "10",
		"--desired-replicas", "3",
		"--cpu-target-utilization", "80",
		"--memory-target-utilization", "75",
		"--cpu-request", "100m",
		"--cpu-limit", "500m",
		"--memory-request", "128Mi",
		"--memory-limit", "512Mi",
		"--scaling-profile", "balanced",
		"--placement-policy", "spread",
		"--canary-enabled",
		"--canary-step-weight", "10",
		"--canary-interval", "1m",
		"--canary-max-weight", "50",
		"--canary-success-threshold", "0.99",
		"--canary-error-threshold", "0.01",
		"--canary-latency-p99-ms", "500",
		"--node-selector", `{"tier":"web"}`,
		"--tolerations", `[{"key":"spot","operator":"Exists","effect":"NoSchedule"}]`,
		"--template-values", `{"replicas":"3"}`,
		"--template-extra-resources", `{"config.yaml":"content"}`,
	}
	if err := sub.ParseFlags(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if err := sub.RunE(sub, []string{}); err != nil {
		t.Fatalf("create dry-run: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal dry-run output: %v\noutput: %s", err, out.String())
	}
	data, _ := body["data"].(map[string]interface{})
	attrs, _ := data["attributes"].(map[string]interface{})

	for _, field := range fields {
		if _, ok := attrs[field]; !ok {
			t.Errorf("dry-run body missing spec field %q in data.attributes", field)
		}
	}
}
