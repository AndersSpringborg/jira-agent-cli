//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBoundaryStore(t *testing.T, privileged bool) *BoundaryStore {
	t.Helper()
	return NewBoundaryStore(filepath.Join(t.TempDir(), "boundaries"), "alice", OwnerGuard{
		TrustedUID: uint32(os.Getuid()),
		Privileged: func() bool { return privileged },
	})
}

func safeBoundary() *Boundary {
	return &Boundary{Projects: []string{"CER"}, Labels: []string{"safe"}}
}

func issueWith(project string, labels ...any) map[string]any {
	return map[string]any{
		"key": project + "-1",
		"fields": map[string]any{
			"project": map[string]any{"key": project},
			"labels":  labels,
		},
	}
}

func TestBoundaryJQLRequiresProjectAndEveryLabel(t *testing.T) {
	b := &Boundary{Projects: []string{"CER", "OPS"}, Labels: []string{"safe", "public"}}

	assert.Equal(t, `project in ("CER", "OPS") AND labels = "safe" AND labels = "public"`, b.JQL())
}

func TestBoundaryJQLEscapesQuotes(t *testing.T) {
	b := &Boundary{Labels: []string{`a"b`}}

	assert.Equal(t, `labels = "a\"b"`, b.JQL())
}

func TestEmptyBoundaryIsInactive(t *testing.T) {
	assert.False(t, (&Boundary{}).Active())
	assert.False(t, (*Boundary)(nil).Active())
	assert.True(t, safeBoundary().Active())
}

func TestBoundaryAllowsOnlyIssuesInsideIt(t *testing.T) {
	b := safeBoundary()

	assert.True(t, b.Allows(issueWith("CER", "safe", "other")))
	assert.False(t, b.Allows(issueWith("CER", "other")), "missing label")
	assert.False(t, b.Allows(issueWith("OPS", "safe")), "other project")
	assert.False(t, b.Allows(map[string]any{"key": "CER-1"}), "fields missing fails closed")
}

func TestBoundaryStoreRejectsUnprivilegedSave(t *testing.T) {
	store := newTestBoundaryStore(t, false)

	err := store.Save(safeBoundary())

	require.ErrorIs(t, err, ErrBoundaryRequiresSudo)
	_, statErr := os.Stat(store.Path())
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestBoundaryStorePrivilegedSaveRoundTripsAndIsReadOnlyForOthers(t *testing.T) {
	store := newTestBoundaryStore(t, true)

	require.NoError(t, store.Save(safeBoundary()))

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, safeBoundary(), loaded)

	fileInfo, err := os.Stat(store.Path())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), fileInfo.Mode().Perm())
	dirInfo, err := os.Stat(filepath.Dir(store.Path()))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), dirInfo.Mode().Perm())
}

func TestBoundaryStoreLoadsInactiveBoundaryWhenNothingWasSaved(t *testing.T) {
	store := newTestBoundaryStore(t, false)

	loaded, err := store.Load()

	require.NoError(t, err)
	assert.False(t, loaded.Active())
}

func TestBoundaryStoreRejectsFileOwnedByUntrustedUser(t *testing.T) {
	store := newTestBoundaryStore(t, true)
	require.NoError(t, store.Save(safeBoundary()))
	store.guard = OwnerGuard{TrustedUID: uint32(os.Getuid()) + 1, Privileged: func() bool { return false }}

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to use it")
}

func TestBoundaryStoreRejectsWritableFile(t *testing.T) {
	store := newTestBoundaryStore(t, true)
	require.NoError(t, store.Save(safeBoundary()))
	require.NoError(t, os.Chmod(store.Path(), 0o666))

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable")
}

func TestBoundaryStoreRejectsWritableDirectoryEvenWithoutFile(t *testing.T) {
	store := newTestBoundaryStore(t, true)
	dir := filepath.Dir(store.Path())
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.Chmod(dir, 0o777))

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable")
}

func TestBoundaryStoreRejectsWritableParentDirectory(t *testing.T) {
	store := newTestBoundaryStore(t, true)
	require.NoError(t, store.Save(safeBoundary()))
	require.NoError(t, os.Chmod(filepath.Dir(filepath.Dir(store.Path())), 0o777))

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable")
}

func TestBoundaryStoreClearRequiresPrivilegeAndRemovesFile(t *testing.T) {
	store := newTestBoundaryStore(t, true)
	require.NoError(t, store.Save(safeBoundary()))

	store.guard = OwnerGuard{TrustedUID: uint32(os.Getuid()), Privileged: func() bool { return false }}
	require.ErrorIs(t, store.Clear(), ErrBoundaryRequiresSudo)

	store.guard = OwnerGuard{TrustedUID: uint32(os.Getuid()), Privileged: func() bool { return true }}
	require.NoError(t, store.Clear())
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.False(t, loaded.Active())
}

func TestBoundaryStorePathIsPerUser(t *testing.T) {
	store := NewBoundaryStore("/etc/jira-cli/boundaries", "alice", OwnerGuard{})

	assert.Equal(t, "/etc/jira-cli/boundaries/alice.yml", store.Path())
}

func TestBoundaryUserUsesSudoUserOnlyWhenRunningAsRoot(t *testing.T) {
	t.Setenv("SUDO_USER", "alice")

	asRoot, err := boundaryUser(true)
	require.NoError(t, err)
	assert.Equal(t, "alice", asRoot)

	asUser, err := boundaryUser(false)
	require.NoError(t, err)
	assert.NotEqual(t, "alice", asUser)
}

func TestBoundaryUserRejectsPathLikeNames(t *testing.T) {
	t.Setenv("SUDO_USER", "../bob")

	_, err := boundaryUser(true)

	require.Error(t, err)
}
