//go:build !windows

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contextHarness struct {
	t          *testing.T
	dir        string
	privileged bool
}

func newContextHarness(t *testing.T) *contextHarness {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Save(&config.Config{
		DefaultProfile: "default",
		Profiles: map[string]*config.Profile{
			"default": {Name: "default"},
			"trifork": {Name: "trifork"},
		},
	}))
	return &contextHarness{t: t, dir: filepath.Join(t.TempDir(), "contexts")}
}

func (h *contextHarness) factory() *cmdutil.Factory {
	privileged := h.privileged
	return &cmdutil.Factory{Contexts: config.NewContextStore(h.dir, "alice", config.OwnerGuard{
		TrustedUID: uint32(os.Getuid()),
		Privileged: func() bool { return privileged },
	})}
}

func (h *contextHarness) run(args ...string) (string, error) {
	h.t.Helper()
	root := newRootCmd("test", "today", h.factory())
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String() + stderr.String(), err
}

func (h *contextHarness) asSudo(args ...string) {
	h.t.Helper()
	h.privileged = true
	defer func() { h.privileged = false }()
	_, err := h.run(args...)
	require.NoError(h.t, err)
}

func TestContextSetWithoutSudoTellsAgentToAskTheUser(t *testing.T) {
	h := newContextHarness(t)

	_, err := h.run("context", "set", "work", "--project", "CER", "--label", "a b")

	require.ErrorIs(t, err, config.ErrContextRequiresSudo)
	assert.Contains(t, err.Error(), "Ask the user to run this command")
	assert.Contains(t, err.Error(), "Never run it yourself")
	assert.Contains(t, err.Error(), "never try to bypass")
	assert.Contains(t, err.Error(), "sudo jira context set work --label 'a b' --project CER")
	_, statErr := os.Stat(filepath.Join(h.dir, "alice.yml"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestContextUseAndClearWithoutSudoAreRejected(t *testing.T) {
	h := newContextHarness(t)
	h.asSudo("context", "set", "work", "--project", "CER")

	_, useErr := h.run("context", "use", "work")
	_, clearErr := h.run("context", "clear", "work")

	require.ErrorIs(t, useErr, config.ErrContextRequiresSudo)
	assert.Contains(t, useErr.Error(), "sudo jira context use work")
	require.ErrorIs(t, clearErr, config.ErrContextRequiresSudo)
	assert.Contains(t, clearErr.Error(), "sudo jira context clear work")
}

func TestContextsSetWithSudoAreReadableWithoutSudo(t *testing.T) {
	h := newContextHarness(t)
	h.asSudo("context", "set", "cai", "--project", "CAI", "--profile", "trifork")
	h.asSudo("context", "set", "personal", "--project", "HOME", "--profile", "default")
	h.asSudo("context", "use", "cai")

	out, err := h.run("context", "show")
	require.NoError(t, err)
	var shown map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &shown))
	assert.Equal(t, "cai", shown["name"])
	assert.Equal(t, "CAI", shown["project"])
	assert.Equal(t, "trifork", shown["profile"])
	assert.Equal(t, true, shown["protected"])
	assert.Equal(t, filepath.Join(h.dir, "alice.yml"), shown["path"])

	out, err = h.run("context", "list")
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"name":"cai","active":true,"profile":"trifork","project":"CAI"},
		{"name":"personal","active":false,"profile":"default","project":"HOME"}
	]`, out)

	profile, err := h.factory().LoadProfile()
	require.NoError(t, err)
	assert.Equal(t, "trifork", profile.Name)
	require.NotNil(t, profile.Context)
	assert.Equal(t, "CAI", profile.Context.Project)
}

func TestContextsInUserConfigAreIgnored(t *testing.T) {
	h := newContextHarness(t)
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "jira-cli", "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(`default_profile: default
active_context: injected
contexts:
  - name: injected
    project: EVIL
profiles:
  default:
    name: default
`), 0o600))

	profile, err := h.factory().LoadProfile()
	require.NoError(t, err)
	assert.Nil(t, profile.Context)

	out, err := h.run("context", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, "EVIL")
	assert.Contains(t, out, "ignored")
}
