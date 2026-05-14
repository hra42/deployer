package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/hra42/deployer/internal/sshx"
	"github.com/hra42/deployer/internal/state"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List services deployed on the host",
	RunE:  runList,
}

func init() {
	listCmd.Flags().String("config", "", "Path to config file (default: $DEPLOYER_CONFIG, ~/.deployer.yml, then /etc/deployer.yml)")
	listCmd.Flags().String("ssh-host", "", "SSH host (user@ip), overrides config")
	listCmd.Flags().String("ssh-key-path", "", "SSH private key path, overrides config")
	listCmd.Flags().String("clone-path", "", "Remote clone base path, overrides config")

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigFromFlag(cmd)
	if err != nil {
		return err
	}

	bindings := []flagBinding{
		{"ssh-host", &cfg.SSHHost},
		{"ssh-key-path", &cfg.SSHKeyPath},
		{"clone-path", &cfg.ClonePath},
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
	defer client.Close()

	st, err := state.Load(cmd.Context(), client, cfg.ClonePath)
	if err != nil {
		return err
	}

	if len(st) == 0 {
		fmt.Println("no services deployed")
		return nil
	}

	domains := make([]string, 0, len(st))
	for d := range st {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	type row struct {
		domain, repo, status, when string
	}
	rows := make([]row, 0, len(domains))
	for _, d := range domains {
		svc := st[d]
		status, err := psStatus(cmd.Context(), client, cfg.ClonePath, svc)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", d, err)
		}
		rows = append(rows, row{d, svc.Repo, status, relativeTime(svc.DeployedAt)})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DOMAIN\tREPO\tSTATUS\tLAST DEPLOY")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.domain, r.repo, r.status, r.when)
	}
	return w.Flush()
}

func psStatus(ctx context.Context, client *sshx.Client, clonePath string, svc state.Service) (string, error) {
	dir := path.Join(clonePath, svc.Slug)
	cmd := fmt.Sprintf(
		"cd %s && COMPOSE_PROJECT_NAME=%s docker compose ps --format json",
		shellQuote(dir), shellQuote(svc.Slug),
	)
	out, err := client.Output(ctx, cmd)
	if err != nil {
		return "", err
	}

	type psEntry struct {
		State string `json:"State"`
	}

	total, running := 0, 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e psEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return "", fmt.Errorf("parse compose ps output: %w", err)
		}
		total++
		if e.State == "running" {
			running++
		}
	}

	switch {
	case total == 0, running == 0:
		return "stopped", nil
	case running == total:
		return "running", nil
	default:
		return "partial", nil
	}
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
