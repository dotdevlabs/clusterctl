package version

import "fmt"

// These are set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

func Info() VersionInfo {
	return VersionInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

func String() string {
	return fmt.Sprintf("clusterctl %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
