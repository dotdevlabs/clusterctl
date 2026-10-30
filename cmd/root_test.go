package cmd

import (
	"bytes"
	"encoding/json"
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
