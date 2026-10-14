package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrBoundaryRequiresSudo is returned when an unprivileged process tries to
// change the protected boundary.
var ErrBoundaryRequiresSudo = errors.New("changing the jira boundary requires sudo (Administrator on Windows)")

// Boundary limits which issues the CLI may see or change. An issue is inside
// it when its project is one of Projects (if any) and it carries every label
// in Labels.
type Boundary struct {
	Projects []string `yaml:"projects,omitempty"`
	Labels   []string `yaml:"labels,omitempty"`
}

func (b *Boundary) Active() bool {
	return b != nil && (len(b.Projects) > 0 || len(b.Labels) > 0)
}

// JQL returns the boundary as a JQL condition.
func (b *Boundary) JQL() string {
	var clauses []string
	if len(b.Projects) > 0 {
		quoted := make([]string, len(b.Projects))
		for i, p := range b.Projects {
			quoted[i] = jqlString(p)
		}
		clauses = append(clauses, fmt.Sprintf("project in (%s)", strings.Join(quoted, ", ")))
	}
	for _, label := range b.Labels {
		clauses = append(clauses, "labels = "+jqlString(label))
	}
	return strings.Join(clauses, " AND ")
}

// Allows reports whether a Jira issue payload is inside the boundary. Missing
// fields count as outside.
func (b *Boundary) Allows(issue map[string]any) bool {
	if !b.Active() {
		return true
	}
	fields, _ := issue["fields"].(map[string]any)
	if len(b.Projects) > 0 {
		project, _ := fields["project"].(map[string]any)
		key, _ := project["key"].(string)
		if !b.AllowsProject(key) {
			return false
		}
	}
	labels, _ := fields["labels"].([]any)
	for _, required := range b.Labels {
		if !containsLabel(labels, required) {
			return false
		}
	}
	return true
}

// AllowsProject reports whether issues in project may be inside the boundary.
func (b *Boundary) AllowsProject(project string) bool {
	if !b.Active() || len(b.Projects) == 0 {
		return true
	}
	for _, p := range b.Projects {
		if strings.EqualFold(p, project) {
			return true
		}
	}
	return false
}

func containsLabel(labels []any, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

func jqlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Guard is the operating-system boundary deciding who may change
// the boundary and which boundary files can be trusted.
type Guard interface {
	IsPrivileged() bool
	// CheckTrusted fails when path could have been written by an
	// unprivileged user.
	CheckTrusted(path string) error
	// Protect restricts path so only privileged users can change it.
	Protect(path string) error
}

// BoundaryStore keeps one user's boundary in a file only privileged users can
// change, so agents running as that user can read but not widen it.
type BoundaryStore struct {
	dir   string
	user  string
	guard Guard
}

func NewBoundaryStore(dir, user string, guard Guard) *BoundaryStore {
	return &BoundaryStore{dir: dir, user: user, guard: guard}
}

// DefaultBoundaryStore returns the system-wide store for the invoking user.
// Under sudo the invoking user is $SUDO_USER, not root.
func DefaultBoundaryStore() (*BoundaryStore, error) {
	guard := defaultGuard()
	name, err := boundaryUser(guard.IsPrivileged())
	if err != nil {
		return nil, err
	}
	return NewBoundaryStore(systemBoundaryDir(), name, guard), nil
}

func (s *BoundaryStore) Path() string {
	return filepath.Join(s.dir, s.user+".yml")
}

// Load reads the boundary. A missing file means no boundary; a file that an
// unprivileged user could have written, replaced, or moved away is rejected.
func (s *BoundaryStore) Load() (*Boundary, error) {
	for _, path := range []string{filepath.Dir(s.dir), s.dir, s.Path()} {
		if err := s.checkTrusted(path); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(s.Path())
	if errors.Is(err, os.ErrNotExist) {
		return &Boundary{}, nil
	}
	if err != nil {
		return nil, err
	}

	var boundary Boundary
	if err := yaml.Unmarshal(data, &boundary); err != nil {
		return nil, fmt.Errorf("parse boundary file %s: %w", s.Path(), err)
	}
	return &boundary, nil
}

func (s *BoundaryStore) checkTrusted(path string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := s.guard.CheckTrusted(path); err != nil {
		return fmt.Errorf("%w; refusing to use it. Ask the user to recreate it with `sudo jira boundary set`", err)
	}
	return nil
}

// RequirePrivilege fails with ErrBoundaryRequiresSudo unless the caller may
// change the boundary.
func (s *BoundaryStore) RequirePrivilege() error {
	if !s.guard.IsPrivileged() {
		return ErrBoundaryRequiresSudo
	}
	return nil
}

// Save atomically replaces the boundary file. Only privileged users may save.
func (s *BoundaryStore) Save(boundary *Boundary) error {
	if err := s.RequirePrivilege(); err != nil {
		return err
	}

	data, err := yaml.Marshal(boundary)
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

// Clear removes the boundary. Only privileged users may clear it.
func (s *BoundaryStore) Clear() error {
	if err := s.RequirePrivilege(); err != nil {
		return err
	}
	if err := os.Remove(s.Path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// boundaryUser names the user whose boundary is in effect. $SUDO_USER is
// honored only when running privileged, so it cannot redirect normal reads.
func boundaryUser(privileged bool) (string, error) {
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
		return "", fmt.Errorf("invalid user name %q for boundary file", name)
	}
	return name, nil
}
