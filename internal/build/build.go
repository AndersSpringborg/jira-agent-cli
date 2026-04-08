// Package build contains build-time metadata injected via ldflags.
package build

var (
	// Version is the semantic version, set at build time via -ldflags.
	Version = "dev"
	// Date is the build date, set at build time via -ldflags.
	Date = "unknown"
)
