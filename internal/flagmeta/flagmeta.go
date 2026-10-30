// Package flagmeta declares the machine-readable flag metadata (allowed values
// and JSON schemas) that "clusterctl ai --json" reports.
package flagmeta

import (
	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/airef"
)

// Closed value sets, shared by the commands and the tests.
var (
	ClusterTypes = []string{"virtual", "imported"}
	SourceTypes  = []string{"helm", "git"}
	// ScalingModes is advisory metadata: the flag help says "e.g." and the
	// client does not validate the value.
	ScalingModes = []string{"hpa", "keda", "manual"}
)

// JSON schemas of JSON-valued flags. They describe the JSON value; flags that
// also accept @file read the file's content as that value.
const (
	StringMapSchema = `{"type":"object","additionalProperties":{"type":"string"}}`
	AnyMapSchema    = `{"type":"object"}`

	TolerationsSchema = `{"type":"array","items":{"type":"object","properties":{` +
		`"key":{"type":"string"},"operator":{"type":"string"},` +
		`"value":{"type":"string"},"effect":{"type":"string"}}}}`

	TemplateInputsSchema = `{"type":"array","items":{"type":"object","required":["key"],"properties":{` +
		`"key":{"type":"string"},"type":{"type":"string"},"default":{},` +
		`"required":{"type":"boolean"},"description":{"type":"string"}}}}`
)

// MustEnum declares the allowed values of flag on cmd; it panics on a
// programming error (unknown flag, empty or duplicate values).
func MustEnum(cmd *cobra.Command, flag string, values []string) {
	if err := airef.SetFlagEnum(cmd, flag, values); err != nil {
		panic(err)
	}
}

// MustSchema declares the JSON schema of flag on cmd; it panics on a
// programming error (unknown flag, invalid schema).
func MustSchema(cmd *cobra.Command, flag, schema string) {
	if err := airef.SetFlagSchema(cmd, flag, schema); err != nil {
		panic(err)
	}
}
