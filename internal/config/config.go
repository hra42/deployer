package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultCNAMETarget = "ai-host.seventhings.app"

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

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".deployer.yml"), nil
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

func Save(path string, c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
