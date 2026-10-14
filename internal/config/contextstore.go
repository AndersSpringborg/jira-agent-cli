package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrContextRequiresSudo is returned when an unprivileged process tries to
// change the protected context file.
var ErrContextRequiresSudo = errors.New("changing the jira context requires sudo (Administrator on Windows)")

// ContextFile is the on-disk schema of a user's protected context file.
type ContextFile struct {
	ActiveContext string     `yaml:"active_context,omitempty"`
	Contexts      []*Context `yaml:"contexts,omitempty"`
}

func (f *ContextFile) Get(name string) *Context {
	for _, ctx := range f.Contexts {
		if ctx != nil && ctx.Name == name {
			return ctx
		}
	}
	return nil
}

func (f *ContextFile) Active() *Context {
	if f.ActiveContext == "" {
		return nil
	}
	return f.Get(f.ActiveContext)
}

func (f *ContextFile) Upsert(name string, ctx *Context) {
	ctx.Name = name
	for i, existing := range f.Contexts {
		if existing != nil && existing.Name == name {
			f.Contexts[i] = ctx
			return
		}
	}
	f.Contexts = append(f.Contexts, ctx)
}

func (f *ContextFile) Delete(name string) bool {
	for i, ctx := range f.Contexts {
		if ctx != nil && ctx.Name == name {
			f.Contexts = append(f.Contexts[:i], f.Contexts[i+1:]...)
			return true
		}
	}
	return false
}

func (f *ContextFile) Names() []string {
	names := make([]string, 0, len(f.Contexts))
	for _, ctx := range f.Contexts {
		if ctx != nil {
			names = append(names, ctx.Name)
		}
	}
	sort.Strings(names)
	return names
}

// ContextGuard is the operating-system boundary deciding who may change
// contexts and which context files can be trusted.
type ContextGuard interface {
	IsPrivileged() bool
	// CheckTrusted fails when path could have been written by an
	// unprivileged user.
	CheckTrusted(path string) error
	// Protect restricts path so only privileged users can change it.
	Protect(path string) error
}

// ContextStore keeps one user's contexts in a file only privileged users can
// change, so agents running as that user can read but not alter their scope.
type ContextStore struct {
	dir   string
	user  string
	guard ContextGuard
}

func NewContextStore(dir, user string, guard ContextGuard) *ContextStore {
	return &ContextStore{dir: dir, user: user, guard: guard}
}

// DefaultContextStore returns the system-wide store for the invoking user.
// Under sudo the invoking user is $SUDO_USER, not root.
func DefaultContextStore() (*ContextStore, error) {
	guard := defaultContextGuard()
	name, err := contextUser(guard.IsPrivileged())
	if err != nil {
		return nil, err
	}
	return NewContextStore(systemContextDir(), name, guard), nil
}

func (s *ContextStore) Path() string {
	return filepath.Join(s.dir, s.user+".yml")
}

// Load reads the contexts. A missing file means no contexts; a file that an
// unprivileged user could have written, replaced, or moved away is rejected.
func (s *ContextStore) Load() (*ContextFile, error) {
	for _, path := range []string{filepath.Dir(s.dir), s.dir, s.Path()} {
		if err := s.checkTrusted(path); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(s.Path())
	if errors.Is(err, os.ErrNotExist) {
		return &ContextFile{}, nil
	}
	if err != nil {
		return nil, err
	}

	var file ContextFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse context file %s: %w", s.Path(), err)
	}
	return &file, nil
}

func (s *ContextStore) checkTrusted(path string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := s.guard.CheckTrusted(path); err != nil {
		return fmt.Errorf("%w; refusing to use it. Recreate it with `sudo jira context set`", err)
	}
	return nil
}

// RequirePrivilege fails with ErrContextRequiresSudo unless the caller may
// change contexts.
func (s *ContextStore) RequirePrivilege() error {
	if !s.guard.IsPrivileged() {
		return ErrContextRequiresSudo
	}
	return nil
}

// Save atomically replaces the context file. Only privileged users may save.
func (s *ContextStore) Save(file *ContextFile) error {
	if err := s.RequirePrivilege(); err != nil {
		return err
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(s.dir, 0o755); err != nil {
		return err
	}
	for _, dir := range []string{filepath.Dir(s.dir), s.dir} {
		if err := s.guard.Protect(dir); err != nil {
			return err
		}
	}

	tmp, err := os.CreateTemp(s.dir, "."+s.user+"-*.yml")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := s.guard.Protect(tmp.Name()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.Path())
}

// contextUser names the user whose contexts are in effect. $SUDO_USER is
// honored only when running privileged, so it cannot redirect normal reads.
func contextUser(privileged bool) (string, error) {
	name := ""
	if privileged {
		name = os.Getenv("SUDO_USER")
	}
	if name == "" {
		current, err := user.Current()
		if err != nil {
			return "", err
		}
		name = current.Username
		// Windows usernames are DOMAIN\user.
		if i := strings.LastIndex(name, `\`); i >= 0 {
			name = name[i+1:]
		}
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid user name %q for context file", name)
	}
	return name, nil
}
