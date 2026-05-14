package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/hra42/deployer/internal/config"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactively create the deployer config (overwrites any existing file)",
	Long: `Write the deployer config interactively.

By default, writes to ~/.deployer.yml (per-user, mode 0600).
With --system, writes to ` + config.SystemPath + ` (shared by all users, mode 0640).
The system path requires root; setup will re-exec under sudo if available.`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().Bool("system", false, "Write the system-wide config at "+config.SystemPath+" (auto-elevates via sudo)")
}

func runSetup(cmd *cobra.Command, args []string) error {
	system, _ := cmd.Flags().GetBool("system")

	path, err := resolveSetupPath(system)
	if err != nil {
		return err
	}

	if system && os.Geteuid() != 0 {
		return reExecWithSudo()
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
	cfg.CloudflareZoneID = prompt(r, "Cloudflare zone ID (blank to auto-detect from domain)", "")
	cfg.CloudflareAccountID = prompt(r, "Cloudflare account ID", "")
	cfg.ZeroTrustPolicyID = prompt(r, "Zero Trust policy ID", "")
	cfg.CNAMETarget = prompt(r, "CNAME target", config.DefaultCNAMETarget)

	if err := config.Save(path, &cfg); err != nil {
		return err
	}
	fmt.Printf("\nWrote %s\n", path)
	if system {
		fmt.Println("Note: file contains secrets. Ensure /etc/deployer.yml is not world-readable")
		fmt.Println("      (default mode 0640, owner root). Consider `chown root:deployer` and adding")
		fmt.Println("      authorized users to the `deployer` group.")
	}
	return nil
}

func resolveSetupPath(system bool) (string, error) {
	if system {
		return config.SystemPath, nil
	}
	return config.UserPath()
}

// reExecWithSudo replaces the current process with `sudo <self> <args...>`.
// Fails loudly if sudo is not on PATH — the user asked for --system and we
// won't silently downgrade to a user-scoped install.
func reExecWithSudo() error {
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("--system requires root and sudo is not available: %w", err)
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate self for sudo re-exec: %w", err)
	}
	fmt.Fprintln(os.Stderr, "elevating to root via sudo for --system setup")
	argv := append([]string{"sudo", "--preserve-env=HOME,USER", self}, os.Args[1:]...)
	return syscall.Exec(sudo, argv, os.Environ())
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
