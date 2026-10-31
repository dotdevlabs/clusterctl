package flagmeta

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

const specPath = "../../docs/api_spec.yaml"

func loadSpec(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// contractEnum returns the values the API contract copy lists under
// x-flag-enums.<key>.
func contractEnum(doc map[string]any, key string) ([]string, error) {
	enums, ok := doc["x-flag-enums"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("x-flag-enums missing or not a map")
	}
	raw, ok := enums[key]
	if !ok {
		return nil, fmt.Errorf("x-flag-enums.%s missing", key)
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("x-flag-enums.%s is not a list", key)
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("x-flag-enums.%s has non-string value %v", key, v)
		}
		out = append(out, s)
	}
	return out, nil
}

func TestEnumsMatchAPIContract(t *testing.T) {
	doc := loadSpec(t)
	for key, want := range map[string][]string{
		"source_type":  SourceTypes,
		"scaling_mode": ScalingModes,
	} {
		got, err := contractEnum(doc, key)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: contract %v, flagmeta %v", key, got, want)
		}
	}
	if !reflect.DeepEqual(SourceTypes, []string{"git", "oci", "helm"}) ||
		!reflect.DeepEqual(ScalingModes, []string{"fixed", "hpa"}) {
		t.Errorf("unexpected enums: %v %v", SourceTypes, ScalingModes)
	}
}

func TestContractEnumErrors(t *testing.T) {
	cases := map[string]map[string]any{
		"missing block": {},
		"missing key":   {"x-flag-enums": map[string]any{}},
		"not a list":    {"x-flag-enums": map[string]any{"k": "git"}},
		"non-string":    {"x-flag-enums": map[string]any{"k": []any{1}}},
	}
	for name, doc := range cases {
		if _, err := contractEnum(doc, "k"); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
