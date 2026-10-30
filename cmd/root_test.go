package cmd

import (
	"bytes"
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/dotdevlabs/ctlkit/pkg/airef"
)

func runAI(t *testing.T, args ...string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return out.String()
}

func findCommand(cmds []airef.CommandRef, path ...string) *airef.CommandRef {
	if len(path) == 0 {
		return nil
	}
	head, rest := path[0], path[1:]
	for i := range cmds {
		if strings.Fields(cmds[i].Use)[0] != head {
			continue
		}
		if len(rest) == 0 {
			return &cmds[i]
		}
		return findCommand(cmds[i].Subcommands, rest...)
	}
	return nil
}

func TestAIJSON(t *testing.T) {
	out := runAI(t, "ai", "--json")
	var ref airef.Reference
	if err := json.Unmarshal([]byte(out), &ref); err != nil {
		t.Fatalf("output is not an airef.Reference: %v\n%s", err, out)
	}
	if ref.Product != "clusterctl" {
		t.Errorf("product = %q", ref.Product)
	}
	if len(ref.Workflows) != len(aiWorkflows()) {
		t.Errorf("workflows = %d, want %d", len(ref.Workflows), len(aiWorkflows()))
	}

	create := findCommand(ref.Commands, "templates", "create")
	if create == nil {
		t.Fatalf("templates create not found in %d commands", len(ref.Commands))
	}
	required, optional := map[string]bool{}, 0
	for _, f := range create.Flags {
		if f.Required {
			required[f.Name] = true
		} else {
			optional++
		}
	}
	if !required["slug"] || !required["name"] {
		t.Errorf("slug/name not marked required: %+v", create.Flags)
	}
	if optional == 0 {
		t.Errorf("expected at least one non-required flag: %+v", create.Flags)
	}
}

func TestAIMarkdown(t *testing.T) {
	out := runAI(t, "ai")
	if json.Valid([]byte(out)) {
		t.Fatalf("expected markdown, got JSON:\n%s", out)
	}
	if !strings.Contains(out, "clusterctl") || !strings.Contains(out, aiWorkflows()[0].Name) {
		t.Errorf("markdown missing product or workflow name:\n%s", out)
	}
}

func aiReference(t *testing.T) airef.Reference {
	t.Helper()
	var ref airef.Reference
	if err := json.Unmarshal([]byte(runAI(t, "ai", "--json")), &ref); err != nil {
		t.Fatalf("decode ai --json: %v", err)
	}
	return ref
}

func findFlag(t *testing.T, ref airef.Reference, flag string, path ...string) airef.FlagRef {
	t.Helper()
	c := findCommand(ref.Commands, path...)
	if c == nil {
		t.Fatalf("command %v not found", path)
	}
	for _, f := range c.Flags {
		if f.Name == flag {
			return f
		}
	}
	t.Fatalf("flag --%s not found on %v", flag, path)
	return airef.FlagRef{}
}

func TestAIFlagEnums(t *testing.T) {
	ref := aiReference(t)
	cases := []struct {
		path []string
		flag string
		want []string
	}{
		{[]string{"clusters", "create"}, "cluster-type", []string{"virtual", "imported"}},
		{[]string{"packages", "create"}, "source-type", []string{"helm", "git"}},
		{[]string{"packages", "update"}, "source-type", []string{"helm", "git"}},
		{[]string{"deployments", "create"}, "scaling-mode", []string{"hpa", "keda", "manual"}},
		{[]string{"deployments", "update"}, "scaling-mode", []string{"hpa", "keda", "manual"}},
	}
	for _, tc := range cases {
		f := findFlag(t, ref, tc.flag, tc.path...)
		if !reflect.DeepEqual(f.Enum, tc.want) {
			t.Errorf("%v --%s enum = %v, want %v", tc.path, tc.flag, f.Enum, tc.want)
		}
	}
}

func TestAIFlagSchemas(t *testing.T) {
	ref := aiReference(t)
	deploy := []string{"deployments", "create"}
	deployUpdate := []string{"deployments", "update"}
	cases := []struct {
		path []string
		flag string
		typ  string
	}{
		{deploy, "node-selector", "object"},
		{deploy, "tolerations", "array"},
		{deploy, "template-values", "object"},
		{deploy, "template-extra-resources", "object"},
		{deployUpdate, "node-selector", "object"},
		{deployUpdate, "tolerations", "array"},
		{deployUpdate, "template-values", "object"},
		{deployUpdate, "template-extra-resources", "object"},
		{[]string{"templates", "create"}, "inputs", "array"},
		{[]string{"templates", "create"}, "manifest-files", "object"},
		{[]string{"templates", "update"}, "inputs", "array"},
		{[]string{"templates", "update"}, "manifest-files", "object"},
	}
	for _, tc := range cases {
		f := findFlag(t, ref, tc.flag, tc.path...)
		var schema map[string]any
		if err := json.Unmarshal(f.Schema, &schema); err != nil {
			t.Errorf("%v --%s schema invalid: %v (%s)", tc.path, tc.flag, err, f.Schema)
			continue
		}
		if schema["type"] != tc.typ {
			t.Errorf("%v --%s schema type = %v, want %s", tc.path, tc.flag, schema["type"], tc.typ)
		}
	}
	inputs := findFlag(t, ref, "inputs", "templates", "create")
	for _, k := range []string{"key", "type", "default", "required", "description"} {
		if !strings.Contains(string(inputs.Schema), `"`+k+`"`) {
			t.Errorf("inputs schema missing property %q: %s", k, inputs.Schema)
		}
	}
	tol := findFlag(t, ref, "tolerations", "deployments", "create")
	for _, k := range []string{"key", "operator", "value", "effect"} {
		if !strings.Contains(string(tol.Schema), `"`+k+`"`) {
			t.Errorf("tolerations schema missing property %q: %s", k, tol.Schema)
		}
	}
}

var (
	enumNameRe  = regexp.MustCompile(`^(status|kind|stage|state)$`)
	valueSetRe  = regexp.MustCompile(`\(\w+(\|\w+)+\)`)
	jsonUsageRe = regexp.MustCompile(`(?i)JSON (object|array)`)
	// --format is a Go template; --tags is a comma list or JSON array.
	notJSONOnly = map[string]bool{"format": true, "tags": true}
)

// missingFlagMetadata walks the reference and reports flags that look like
// closed-set or JSON-valued flags but declare no enum or schema.
func missingFlagMetadata(cmds []airef.CommandRef, prefix string) []string {
	var missing []string
	for _, c := range cmds {
		path := prefix + " " + strings.Fields(c.Use)[0]
		for _, f := range c.Flags {
			if notJSONOnly[f.Name] {
				continue
			}
			if (enumNameRe.MatchString(f.Name) || valueSetRe.MatchString(f.Usage)) && len(f.Enum) == 0 {
				missing = append(missing, path+" --"+f.Name+": enum")
			}
			if jsonUsageRe.MatchString(f.Usage) && len(f.Schema) == 0 {
				missing = append(missing, path+" --"+f.Name+": schema")
			}
		}
		missing = append(missing, missingFlagMetadata(c.Subcommands, path)...)
	}
	return missing
}

func TestAIFlagMetadataComplete(t *testing.T) {
	ref := aiReference(t)
	if missing := missingFlagMetadata(ref.Commands, ""); len(missing) > 0 {
		t.Errorf("flags missing enum/schema: %v", missing)
	}

	synthetic := []airef.CommandRef{{Use: "demo", Flags: []airef.FlagRef{
		{Name: "status", Usage: "Status"},
		{Name: "mode", Usage: "Mode (a|b)"},
		{Name: "body", Usage: "Body as a JSON object"},
		{Name: "format", Usage: "Go template (a|b)"},
		{Name: "ok", Usage: "Kind (x|y)", Enum: []string{"x", "y"}},
	}}}
	got := missingFlagMetadata(synthetic, "")
	want := []string{" demo --status: enum", " demo --mode: enum", " demo --body: schema"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("guard reported %v, want %v", got, want)
	}
}

func TestAIMarkdownShowsAllowedValues(t *testing.T) {
	out := runAI(t, "ai")
	if !strings.Contains(out, "Allowed: virtual, imported") {
		t.Errorf("markdown missing allowed values for --cluster-type:\n%s", out)
	}
}
