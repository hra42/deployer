package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultCNAMETarget = "host.example.com"

	// SystemPath is the system-wide config location. Intended to be readable
	// by a deployer group (mode 0640, root:deployer) so multiple users on the
	// same machine can share credentials without each one re-running setup.
	SystemPath = "/etc/deployer.yml"

	// EnvConfigPath, if set, overrides all default search behavior.
	EnvConfigPath = "DEPLOYER_CONFIG"
)

type Config struct {
	SSHHost             string `yaml:"ssh_host"`
	SSHKeyPath          string `yaml:"ssh_key_path"`
	GitHubToken         string `yaml:"github_token"`
	ClonePath           string `yaml:"clone_path"`
	TraefikNetwork      string `yaml:"traefik_network"`
	CloudflareAPIToken  string `yaml:"cloudflare_api_token"`
	CloudflareZoneID    string `yaml:"cloudflare_zone_id"`
	CloudflareAccountID string `yaml:"cloudflare_account_id"`
	ZeroTrustPolicyID   string `yaml:"zero_trust_policy_id"`
	CNAMETarget         string `yaml:"cname_target"`
}

// UserPath returns the per-user config path (~/.deployer.yml).
func UserPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".deployer.yml"), nil
}

// DefaultPath returns the user config path. Retained for callers that want
// the write destination for a per-user setup.
func DefaultPath() (string, error) {
	return UserPath()
}

// ResolveReadPath picks the config file to load. Order of precedence:
//  1. $DEPLOYER_CONFIG (if set; must exist)
//  2. ~/.deployer.yml (if it exists)
//  3. /etc/deployer.yml (if it exists)
//
// If none exist, the user path is returned along with os.ErrNotExist so the
// caller can surface a useful "run setup" message.
func ResolveReadPath() (string, error) {
	if p := os.Getenv(EnvConfigPath); p != "" {
		if _, err := os.Stat(p); err != nil {
			return p, fmt.Errorf("%s=%s: %w", EnvConfigPath, p, err)
		}
		return p, nil
	}
	user, err := UserPath()
	if err == nil {
		if _, statErr := os.Stat(user); statErr == nil {
			return user, nil
		}
	}
	if _, statErr := os.Stat(SystemPath); statErr == nil {
		return SystemPath, nil
	}
	if err != nil {
		return "", err
	}
	return user, os.ErrNotExist
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s — run `deployer setup` first", path)
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &c, nil
}

// Save writes the config. The file mode depends on the path:
// system-wide config gets 0640 (root:deployer-readable), per-user gets 0600.
func Save(path string, c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	mode := os.FileMode(0600)
	if path == SystemPath {
		mode = 0640
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
