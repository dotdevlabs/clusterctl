package version_test

import (
	"strings"
	"testing"

	"github.com/dotdevlabs/clusterctl/internal/version"
)

func TestString(t *testing.T) {
	s := version.String()
	if !strings.HasPrefix(s, "clusterctl ") {
		t.Errorf("unexpected version string: %s", s)
	}
}

func TestInfo(t *testing.T) {
	info := version.Info()
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
}
