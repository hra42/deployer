package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hra42/deployer/internal/config"
	"github.com/hra42/deployer/internal/deploy"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a service from a GitHub repo to the configured host",
	RunE:  runDeploy,
}

type flagBinding struct {
	name  string
	field *string
}

func init() {
	deployCmd.Flags().String("repo", "", "GitHub repo (e.g. github.com/owner/name) (required)")
	deployCmd.Flags().String("domain", "", "Public domain for this deployment (required)")

	deployCmd.Flags().String("ssh-host", "", "SSH host (user@ip), overrides config")
	deployCmd.Flags().String("ssh-key-path", "", "SSH private key path, overrides config")
	deployCmd.Flags().String("github-token", "", "GitHub token, overrides config")
	deployCmd.Flags().String("clone-path", "", "Remote clone base path, overrides config")
	deployCmd.Flags().String("traefik-network", "", "Traefik docker network, overrides config")
	deployCmd.Flags().String("cloudflare-api-token", "", "Cloudflare API token, overrides config")
	deployCmd.Flags().String("cloudflare-zone-id", "", "Cloudflare zone ID, overrides config")
	deployCmd.Flags().String("cloudflare-account-id", "", "Cloudflare account ID, overrides config")
	deployCmd.Flags().String("zero-trust-policy-id", "", "Zero Trust policy ID, overrides config")
	deployCmd.Flags().String("cname-target", "", "CNAME target, overrides config")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	bindings := []flagBinding{
		{"ssh-host", &cfg.SSHHost},
		{"ssh-key-path", &cfg.SSHKeyPath},
		{"github-token", &cfg.GitHubToken},
		{"clone-path", &cfg.ClonePath},
		{"traefik-network", &cfg.TraefikNetwork},
		{"cloudflare-api-token", &cfg.CloudflareAPIToken},
		{"cloudflare-zone-id", &cfg.CloudflareZoneID},
		{"cloudflare-account-id", &cfg.CloudflareAccountID},
		{"zero-trust-policy-id", &cfg.ZeroTrustPolicyID},
		{"cname-target", &cfg.CNAMETarget},
	}
	for _, b := range bindings {
		if cmd.Flags().Changed(b.name) {
			v, _ := cmd.Flags().GetString(b.name)
			*b.field = v
		}
	}

	repo, _ := cmd.Flags().GetString("repo")
	domain, _ := cmd.Flags().GetString("domain")

	required := []struct{ name, val string }{
		{"--repo", repo},
		{"--domain", domain},
		{"ssh_host", cfg.SSHHost},
		{"ssh_key_path", cfg.SSHKeyPath},
		{"github_token", cfg.GitHubToken},
		{"clone_path", cfg.ClonePath},
		{"traefik_network", cfg.TraefikNetwork},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("%s is required", r.name)
		}
	}

	return deploy.Run(cmd.Context(), deploy.Options{
		Repo:   repo,
		Domain: domain,
		Cfg:    cfg,
	})
}
