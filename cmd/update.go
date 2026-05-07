package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hra42/deployer/internal/config"
	"github.com/hra42/deployer/internal/deploy"
	"github.com/hra42/deployer/internal/sshx"
	"github.com/hra42/deployer/internal/state"
)

var updateCmd = &cobra.Command{
	Use:   "update <domain>",
	Short: "Re-deploy a service recorded in remote state",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().String("ssh-host", "", "SSH host (user@ip), overrides config")
	updateCmd.Flags().String("ssh-key-path", "", "SSH private key path, overrides config")
	updateCmd.Flags().String("github-token", "", "GitHub token, overrides config")
	updateCmd.Flags().String("clone-path", "", "Remote clone base path, overrides config")
	updateCmd.Flags().String("traefik-network", "", "Traefik docker network, overrides config")
	updateCmd.Flags().String("cloudflare-api-token", "", "Cloudflare API token, overrides config")
	updateCmd.Flags().String("cloudflare-zone-id", "", "Cloudflare zone ID, overrides config")
	updateCmd.Flags().String("cloudflare-account-id", "", "Cloudflare account ID, overrides config")
	updateCmd.Flags().String("zero-trust-policy-id", "", "Zero Trust policy ID, overrides config")
	updateCmd.Flags().String("cname-target", "", "CNAME target, overrides config")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	domain := args[0]

	cfgPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(cfgPath)
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

	required := []struct{ name, val string }{
		{"ssh_host", cfg.SSHHost},
		{"ssh_key_path", cfg.SSHKeyPath},
		{"clone_path", cfg.ClonePath},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("%s is not set; run `deployer setup` first", r.name)
		}
	}

	client, err := sshx.Dial(cfg.SSHHost, cfg.SSHKeyPath)
	if err != nil {
		return err
	}
	st, err := state.Load(cmd.Context(), client, cfg.ClonePath)
	client.Close()
	if err != nil {
		return err
	}

	svc, ok := st[domain]
	if !ok {
		return fmt.Errorf("service %s not found in state; run `deployer deploy --repo ... --domain %s` first", domain, domain)
	}

	return deploy.Run(cmd.Context(), deploy.Options{
		Repo:   svc.Repo,
		Domain: domain,
		Cfg:    cfg,
	})
}
