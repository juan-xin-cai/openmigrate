package buildinfo

import "strings"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func Summary() string {
	return strings.Join([]string{
		"openmigrate " + clean(Version, "dev"),
		"commit: " + clean(Commit, "none"),
		"built: " + clean(BuildDate, "unknown"),
	}, "\n")
}

func clean(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
