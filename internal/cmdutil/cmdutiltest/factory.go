// Package cmdutiltest provides command factories for tests.
package cmdutiltest

import (
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"
)

// NewFactory returns a Factory with an empty boundary store, so tests never
// pick up the boundary configured on the machine running them.
func NewFactory(t testing.TB) *cmdutil.Factory {
	t.Helper()
	return &cmdutil.Factory{
		Boundaries: config.NewBoundaryStore(t.TempDir(), "tester", trustingGuard{}),
	}
}

// trustingGuard is an unprivileged guard that trusts every file; the store
// directory is a fresh temp dir, so nothing untrusted can be in it.
type trustingGuard struct{}

func (trustingGuard) IsPrivileged() bool        { return false }
func (trustingGuard) CheckTrusted(string) error { return nil }
func (trustingGuard) Protect(string) error      { return nil }
