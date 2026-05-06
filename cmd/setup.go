package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hra42/deployer/internal/config"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactively create ~/.deployer.yml (overwrites any existing file)",
	RunE:  runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}

	fmt.Printf("Writing config to %s (will overwrite if it exists).\n", path)
	fmt.Println("Press enter to accept the default shown in [brackets].")
	fmt.Println()

	r := bufio.NewReader(os.Stdin)
	cfg := config.Config{}

	cfg.SSHHost = prompt(r, "SSH host (user@ip)", "")
	cfg.SSHKeyPath = prompt(r, "SSH private key path", "")
	cfg.GitHubToken = prompt(r, "GitHub token", "")
	cfg.ClonePath = prompt(r, "Remote clone base path", "")
	cfg.TraefikNetwork = prompt(r, "Traefik docker network name", "")
	cfg.CloudflareAPIToken = prompt(r, "Cloudflare API token (blank to skip DNS/ZT)", "")
	cfg.CloudflareZoneID = prompt(r, "Cloudflare zone ID", "")
	cfg.CloudflareAccountID = prompt(r, "Cloudflare account ID", "")
	cfg.ZeroTrustPolicyID = prompt(r, "Zero Trust policy ID", "")
	cfg.CNAMETarget = prompt(r, "CNAME target", config.DefaultCNAMETarget)

	if err := config.Save(path, &cfg); err != nil {
		return err
	}
	fmt.Printf("\nWrote %s\n", path)
	return nil
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return def
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return def
	}
	return line
}
