package config

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultProfile = "default"

type Context struct {
	Name      string   `yaml:"name,omitempty"`
	Profile   string   `yaml:"profile,omitempty"`
	Project   string   `yaml:"project,omitempty"`
	BoardID   int      `yaml:"board_id,omitempty"`
	Epic      string   `yaml:"epic,omitempty"`
	Labels    []string `yaml:"labels,omitempty"`
	IssueType string   `yaml:"issue_type,omitempty"`
	Status    string   `yaml:"status,omitempty"`
	Assignee  string   `yaml:"assignee,omitempty"`
	Display   string   `yaml:"display,omitempty"`
}

type Profile struct {
	Name           string  `yaml:"name"`
	BaseURL        string  `yaml:"base_url,omitempty"`
	AuthType       string  `yaml:"auth_type,omitempty"`
	UserEmail      string  `yaml:"user_email,omitempty"`
	TimeoutSeconds float64 `yaml:"timeout_seconds,omitempty"`
	// Context is the active protected context, filled in at runtime. It is
	// never read from or written to config.yml, which agents can edit.
	Context *Context `yaml:"-"`
}

func DetectAuthType(baseURL string) string {
	if strings.Contains(strings.ToLower(baseURL), ".atlassian.net") {
		return "basic"
	}
	return "pat"
}

type Config struct {
	DefaultProfile string              `yaml:"default_profile"`
	Profiles       map[string]*Profile `yaml:"profiles"`
}

// sudoUser returns the user who invoked sudo, or "" when not running as root
// through sudo.
func sudoUser(euid int) string {
	if euid != 0 {
		return ""
	}
	return os.Getenv("SUDO_USER")
}

// homeDir returns the home of the sudo-invoking user, so `sudo jira context`
// sees the same profiles as the user; otherwise the current user's home.
func homeDir(invokingUser string) (string, error) {
	if invokingUser == "" {
		return os.UserHomeDir()
	}
	u, err := user.Lookup(invokingUser)
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func configDir() (string, error) {
	home, err := homeDir(sudoUser(os.Geteuid()))
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "jira-cli"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{
				DefaultProfile: DefaultProfile,
				Profiles:       make(map[string]*Profile),
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = DefaultProfile
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	if sudoUser(os.Geteuid()) != "" {
		return errors.New("refusing to write config.yml as root; run this command without sudo")
	}
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func ResolveProfileName(cfg *Config, override string) string {
	if override != "" {
		return override
	}
	if envProfile := os.Getenv("JIRABOT_PROFILE"); envProfile != "" {
		return envProfile
	}
	if cfg.DefaultProfile != "" {
		return cfg.DefaultProfile
	}
	return DefaultProfile
}

func GetProfile(cfg *Config, name string) *Profile {
	if cfg.Profiles == nil {
		return nil
	}
	return cfg.Profiles[name]
}

func UpsertProfile(cfg *Config, p *Profile) {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	cfg.Profiles[p.Name] = p
}

func DeleteProfile(cfg *Config, name string) bool {
	if cfg.Profiles == nil {
		return false
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return false
	}
	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = DefaultProfile
	}
	return true
}

func ListProfiles(cfg *Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HasLegacyContexts reports whether config.yml still holds contexts from
// before they moved to the protected context file. They are ignored.
func HasLegacyContexts() bool {
	path, err := configPath()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var legacy struct {
		ActiveContext string           `yaml:"active_context"`
		Contexts      []map[string]any `yaml:"contexts"`
		Profiles      map[string]struct {
			Context map[string]any `yaml:"context"`
		} `yaml:"profiles"`
	}
	if yaml.Unmarshal(data, &legacy) != nil {
		return false
	}
	if legacy.ActiveContext != "" || len(legacy.Contexts) > 0 {
		return true
	}
	for _, p := range legacy.Profiles {
		if len(p.Context) > 0 {
			return true
		}
	}
	return false
}
