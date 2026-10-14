package config

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadIgnoresContextsWrittenToUserConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := configPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`default_profile: default
active_context: injected
contexts:
  - name: injected
    project: EVIL
profiles:
  default:
    name: default
    context:
      project: EVIL
`), 0o600))

	cfg, err := Load()

	require.NoError(t, err)
	assert.Nil(t, GetProfile(cfg, "default").Context)
	assert.True(t, HasLegacyContexts())
}

func TestSudoUserIsOnlyHonoredForRoot(t *testing.T) {
	t.Setenv("SUDO_USER", "alice")

	assert.Equal(t, "alice", sudoUser(0))
	assert.Empty(t, sudoUser(501))
}

func TestHomeDirUnderSudoIsTheInvokingUsersHome(t *testing.T) {
	current, err := user.Current()
	require.NoError(t, err)
	t.Setenv("HOME", t.TempDir())

	home, err := homeDir(current.Username)

	require.NoError(t, err)
	assert.Equal(t, current.HomeDir, home)
}
