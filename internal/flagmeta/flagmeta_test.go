package flagmeta

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/airef"
)

func newCmd() *cobra.Command {
	c := &cobra.Command{Use: "x"}
	c.Flags().String("f", "", "")
	return c
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	fn()
}

func TestMustEnumAndSchema(t *testing.T) {
	c := newCmd()
	MustEnum(c, "f", []string{"a", "b"})
	MustSchema(c, "f", AnyMapSchema)
	ann := c.Flags().Lookup("f").Annotations
	if len(ann[airef.EnumAnnotation]) == 0 || len(ann[airef.SchemaAnnotation]) == 0 {
		t.Errorf("annotations not set: %v", ann)
	}
}

func TestPanics(t *testing.T) {
	mustPanic(t, "unknown enum flag", func() { MustEnum(newCmd(), "nope", []string{"a"}) })
	mustPanic(t, "empty enum", func() { MustEnum(newCmd(), "f", nil) })
	mustPanic(t, "duplicate enum", func() { MustEnum(newCmd(), "f", []string{"a", "a"}) })
	mustPanic(t, "unknown schema flag", func() { MustSchema(newCmd(), "nope", AnyMapSchema) })
	mustPanic(t, "invalid schema", func() { MustSchema(newCmd(), "f", "{") })
}

func TestSchemasAreValid(t *testing.T) {
	for _, s := range []string{StringMapSchema, AnyMapSchema, TolerationsSchema, TemplateInputsSchema} {
		MustSchema(newCmd(), "f", s)
	}
}
