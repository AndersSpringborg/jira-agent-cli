//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContextStore(t *testing.T, privileged bool) *ContextStore {
	t.Helper()
	return NewContextStore(filepath.Join(t.TempDir(), "contexts"), "alice", OwnerGuard{
		TrustedUID: uint32(os.Getuid()),
		Privileged: func() bool { return privileged },
	})
}

func savedContexts() *ContextFile {
	file := &ContextFile{ActiveContext: "work"}
	file.Upsert("work", &Context{Profile: "default", Project: "CER"})
	return file
}

func TestContextStoreRejectsUnprivilegedSave(t *testing.T) {
	store := newTestContextStore(t, false)

	err := store.Save(savedContexts())

	require.ErrorIs(t, err, ErrContextRequiresSudo)
	_, statErr := os.Stat(store.Path())
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestContextStorePrivilegedSaveRoundTripsAndIsReadOnlyForOthers(t *testing.T) {
	store := newTestContextStore(t, true)

	require.NoError(t, store.Save(savedContexts()))

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "work", loaded.ActiveContext)
	require.NotNil(t, loaded.Active())
	assert.Equal(t, "CER", loaded.Active().Project)

	fileInfo, err := os.Stat(store.Path())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), fileInfo.Mode().Perm())
	dirInfo, err := os.Stat(filepath.Dir(store.Path()))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), dirInfo.Mode().Perm())
}

func TestContextStoreLoadsEmptyContextsWhenNothingWasSaved(t *testing.T) {
	store := newTestContextStore(t, false)

	loaded, err := store.Load()

	require.NoError(t, err)
	assert.Nil(t, loaded.Active())
	assert.Empty(t, loaded.Names())
}

func TestContextStoreRejectsFileOwnedByUntrustedUser(t *testing.T) {
	store := newTestContextStore(t, true)
	require.NoError(t, store.Save(savedContexts()))
	store.guard = OwnerGuard{TrustedUID: uint32(os.Getuid()) + 1, Privileged: func() bool { return false }}

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to use it")
}

func TestContextStoreRejectsWritableContextFile(t *testing.T) {
	store := newTestContextStore(t, true)
	require.NoError(t, store.Save(savedContexts()))
	require.NoError(t, os.Chmod(store.Path(), 0o666))

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable")
}

func TestContextStoreRejectsWritableDirectoryEvenWithoutFile(t *testing.T) {
	store := newTestContextStore(t, true)
	dir := filepath.Dir(store.Path())
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.Chmod(dir, 0o777))

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable")
}

func TestContextStorePathIsPerUser(t *testing.T) {
	store := NewContextStore("/etc/jira-cli/contexts", "alice", OwnerGuard{})

	assert.Equal(t, "/etc/jira-cli/contexts/alice.yml", store.Path())
}

func TestContextUserUsesSudoUserOnlyWhenRunningAsRoot(t *testing.T) {
	t.Setenv("SUDO_USER", "alice")

	asRoot, err := contextUser(true)
	require.NoError(t, err)
	assert.Equal(t, "alice", asRoot)

	asUser, err := contextUser(false)
	require.NoError(t, err)
	assert.NotEqual(t, "alice", asUser)
}

func TestContextUserRejectsPathLikeNames(t *testing.T) {
	t.Setenv("SUDO_USER", "../bob")

	_, err := contextUser(true)

	require.Error(t, err)
}

func TestContextStoreRejectsWritableParentDirectory(t *testing.T) {
	store := newTestContextStore(t, true)
	require.NoError(t, store.Save(savedContexts()))
	require.NoError(t, os.Chmod(filepath.Dir(filepath.Dir(store.Path())), 0o777))

	_, err := store.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable")
}
