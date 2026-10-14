package config

import (
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
