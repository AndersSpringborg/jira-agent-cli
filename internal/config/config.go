package config

import (
	"errors"
	"os"
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

func (c *Context) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Profile == "" && c.Project == "" && c.BoardID == 0 && c.Epic == "" &&
		len(c.Labels) == 0 && c.IssueType == "" && c.Status == "" && c.Assignee == "" &&
		c.Display == ""
}

type Profile struct {
	Name           string   `yaml:"name"`
	BaseURL        string   `yaml:"base_url,omitempty"`
	AuthType       string   `yaml:"auth_type,omitempty"`
	UserEmail      string   `yaml:"user_email,omitempty"`
	TimeoutSeconds float64  `yaml:"timeout_seconds,omitempty"`
	Context        *Context `yaml:"context,omitempty"`
}

func DetectAuthType(baseURL string) string {
	if strings.Contains(strings.ToLower(baseURL), ".atlassian.net") {
		return "basic"
	}
	return "pat"
}

type Config struct {
	DefaultProfile string              `yaml:"default_profile"`
	ActiveContext  string              `yaml:"active_context,omitempty"`
	Profiles       map[string]*Profile `yaml:"profiles"`
	Contexts       []*Context          `yaml:"contexts,omitempty"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
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

func GetContext(cfg *Config, name string) *Context {
	if cfg == nil {
		return nil
	}
	for _, ctx := range cfg.Contexts {
		if ctx != nil && ctx.Name == name {
			return ctx
		}
	}
	return nil
}

func GetActiveContext(cfg *Config) *Context {
	if cfg == nil || cfg.ActiveContext == "" {
		return nil
	}
	return GetContext(cfg, cfg.ActiveContext)
}

func UpsertContext(cfg *Config, name string, ctx *Context) {
	ctx.Name = name
	for i, existing := range cfg.Contexts {
		if existing != nil && existing.Name == name {
			cfg.Contexts[i] = ctx
			return
		}
	}
	cfg.Contexts = append(cfg.Contexts, ctx)
}

func DeleteContext(cfg *Config, name string) bool {
	for i, ctx := range cfg.Contexts {
		if ctx != nil && ctx.Name == name {
			cfg.Contexts = append(cfg.Contexts[:i], cfg.Contexts[i+1:]...)
			return true
		}
	}
	return false
}

func ListContexts(cfg *Config) []string {
	names := make([]string, 0, len(cfg.Contexts))
	for _, ctx := range cfg.Contexts {
		if ctx != nil {
			names = append(names, ctx.Name)
		}
	}
	sort.Strings(names)
	return names
}
