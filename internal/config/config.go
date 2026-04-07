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
	return c.Project == "" && c.BoardID == 0 && c.Epic == "" &&
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
	Profiles       map[string]*Profile `yaml:"profiles"`
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
