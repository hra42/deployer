package deploy

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/hra42/deployer/internal/cloudflare"
	"github.com/hra42/deployer/internal/config"
	"github.com/hra42/deployer/internal/slug"
	"github.com/hra42/deployer/internal/sshx"
	"github.com/hra42/deployer/internal/state"
	"github.com/hra42/deployer/internal/ui"
)

type Options struct {
	Repo   string
	Domain string
	Cfg    *config.Config
}

var phaseNames = []string{
	"Connect to host",
	"Sync repo",
	"Validate compose files",
	"Bring up containers",
	"Cloudflare DNS",
	"Cloudflare Zero Trust",
	"Record state",
}

func Run(ctx context.Context, opts Options) (err error) {
	s := slug.Domain(opts.Domain)
	remotePath := path.Join(opts.Cfg.ClonePath, s)

	results := make([]ui.SummaryItem, len(phaseNames))
	for i, name := range phaseNames {
		results[i] = ui.SummaryItem{Name: name, Status: ui.StatusSkipRun}
	}

	completed := false
	defer func() {
		ui.Summary(results)
		if err == nil && completed {
			ui.OK(fmt.Sprintf("deploy complete: %s", opts.Domain))
		}
	}()

	mark := func(idx int, status ui.Status, detail string) {
		results[idx] = ui.SummaryItem{Name: phaseNames[idx], Status: status, Detail: detail}
	}

	// Phase 1: Connect to host
	ui.Phase(1, phaseNames[0])
	client, err := sshx.Dial(opts.Cfg.SSHHost, opts.Cfg.SSHKeyPath)
	if err != nil {
		ui.Failf("ssh %s: %v", opts.Cfg.SSHHost, err)
		mark(0, ui.StatusFailed, fmt.Sprintf("ssh %s", opts.Cfg.SSHHost))
		return err
	}
	defer client.Close()
	ui.OK(fmt.Sprintf("connected to %s", opts.Cfg.SSHHost))
	mark(0, ui.StatusOK, opts.Cfg.SSHHost)

	// Phase 2: Sync repo
	ui.Phase(2, phaseNames[1])
	out, _ := client.Output(ctx, fmt.Sprintf("test -d %s/.git && echo yes || echo no", shellQuote(remotePath)))
	if strings.TrimSpace(out) == "yes" {
		ui.Info(fmt.Sprintf("git pull in %s", remotePath))
		if err = client.Run(ctx, fmt.Sprintf("git -C %s pull --ff-only", shellQuote(remotePath))); err != nil {
			ui.Failf("git pull in %s: %v", remotePath, err)
			mark(1, ui.StatusFailed, fmt.Sprintf("git pull in %s", remotePath))
			return err
		}
	} else {
		ui.Info(fmt.Sprintf("git clone into %s", remotePath))
		var cloneCmd string
		cloneCmd, err = buildCloneCmd(opts.Repo, opts.Cfg.GitHubToken, remotePath, opts.Cfg.ClonePath)
		if err != nil {
			ui.Failf("%v", err)
			mark(1, ui.StatusFailed, err.Error())
			return err
		}
		if err = client.Run(ctx, cloneCmd); err != nil {
			ui.Failf("git clone into %s: %v", remotePath, err)
			mark(1, ui.StatusFailed, fmt.Sprintf("git clone into %s", remotePath))
			return err
		}
	}
	ui.OK("repo synced")
	mark(1, ui.StatusOK, remotePath)

	// Phase 3: Validate compose files
	ui.Phase(3, phaseNames[2])
	for _, f := range []string{"Dockerfile", "docker-compose.yml"} {
		check := fmt.Sprintf("test -f %s/%s", shellQuote(remotePath), f)
		if _, cerr := client.Output(ctx, check); cerr != nil {
			msg := fmt.Sprintf("%s missing in %s", f, remotePath)
			ui.Failf("%s", msg)
			mark(2, ui.StatusFailed, msg)
			err = fmt.Errorf("%s", msg)
			return err
		}
	}
	ui.OK("Dockerfile and docker-compose.yml present")
	mark(2, ui.StatusOK, "")

	// Phase 4: Bring up containers
	ui.Phase(4, phaseNames[3])
	upCmd := fmt.Sprintf(
		"cd %s && COMPOSE_PROJECT_NAME=%s docker compose pull && COMPOSE_PROJECT_NAME=%s docker compose up -d --remove-orphans",
		shellQuote(remotePath), shellQuote(s), shellQuote(s),
	)
	if err = client.Run(ctx, upCmd); err != nil {
		ui.Failf("docker compose (project %s): %v", s, err)
		mark(3, ui.StatusFailed, fmt.Sprintf("project %s", s))
		return err
	}
	ui.OK(fmt.Sprintf("project %s up", s))
	mark(3, ui.StatusOK, fmt.Sprintf("project %s", s))

	// Phase 5: Cloudflare DNS
	ui.Phase(5, phaseNames[4])
	if opts.Cfg.CloudflareAPIToken == "" || opts.Cfg.CloudflareZoneID == "" {
		ui.Warn("skipped: cloudflare credentials not set")
		mark(4, ui.StatusSkipped, "cloudflare credentials not set")
	} else if opts.Cfg.CNAMETarget == "" {
		ui.Warn("skipped: cname_target not set")
		mark(4, ui.StatusSkipped, "cname_target not set")
	} else {
		cf := cloudflare.New(opts.Cfg.CloudflareAPIToken)
		var existing *cloudflare.DNSRecord
		existing, err = cf.FindCNAME(ctx, opts.Cfg.CloudflareZoneID, opts.Domain)
		if err != nil {
			ui.Failf("cloudflare dns for %s: %v", opts.Domain, err)
			mark(4, ui.StatusFailed, fmt.Sprintf("cloudflare dns for %s", opts.Domain))
			return err
		}
		if existing == nil {
			ui.Info(fmt.Sprintf("creating CNAME %s → %s", opts.Domain, opts.Cfg.CNAMETarget))
			if err = cf.CreateCNAME(ctx, opts.Cfg.CloudflareZoneID, opts.Domain, opts.Cfg.CNAMETarget); err != nil {
				ui.Failf("cloudflare dns for %s: %v", opts.Domain, err)
				mark(4, ui.StatusFailed, fmt.Sprintf("cloudflare dns for %s", opts.Domain))
				return err
			}
		} else {
			ui.Info(fmt.Sprintf("updating CNAME %s → %s", opts.Domain, opts.Cfg.CNAMETarget))
			if err = cf.UpdateCNAME(ctx, opts.Cfg.CloudflareZoneID, existing.ID, opts.Domain, opts.Cfg.CNAMETarget); err != nil {
				ui.Failf("cloudflare dns for %s: %v", opts.Domain, err)
				mark(4, ui.StatusFailed, fmt.Sprintf("cloudflare dns for %s", opts.Domain))
				return err
			}
		}
		ui.OK(fmt.Sprintf("proxied CNAME %s → %s", opts.Domain, opts.Cfg.CNAMETarget))
		mark(4, ui.StatusOK, fmt.Sprintf("CNAME %s → %s", opts.Domain, opts.Cfg.CNAMETarget))
	}

	// Phase 6: Cloudflare Zero Trust
	ui.Phase(6, phaseNames[5])
	if opts.Cfg.CloudflareAPIToken == "" || opts.Cfg.CloudflareAccountID == "" {
		ui.Warn("skipped: cloudflare account credentials not set")
		mark(5, ui.StatusSkipped, "cloudflare account credentials not set")
	} else if opts.Cfg.ZeroTrustPolicyID == "" {
		ui.Warn("skipped: zero_trust_policy_id not set")
		mark(5, ui.StatusSkipped, "zero_trust_policy_id not set")
	} else {
		cf := cloudflare.New(opts.Cfg.CloudflareAPIToken)
		var existing *cloudflare.AccessApp
		existing, err = cf.FindAccessApp(ctx, opts.Cfg.CloudflareAccountID, opts.Domain)
		if err != nil {
			ui.Failf("cloudflare access for %s: %v", opts.Domain, err)
			mark(5, ui.StatusFailed, fmt.Sprintf("cloudflare access for %s", opts.Domain))
			return err
		}
		var appID string
		if existing == nil {
			ui.Info(fmt.Sprintf("creating Access app for %s", opts.Domain))
			var created *cloudflare.AccessApp
			created, err = cf.CreateAccessApp(ctx, opts.Cfg.CloudflareAccountID, opts.Domain, opts.Domain)
			if err != nil {
				ui.Failf("cloudflare access for %s: %v", opts.Domain, err)
				mark(5, ui.StatusFailed, fmt.Sprintf("cloudflare access for %s", opts.Domain))
				return err
			}
			appID = created.ID
		} else {
			ui.Info(fmt.Sprintf("updating Access app for %s", opts.Domain))
			if err = cf.UpdateAccessApp(ctx, opts.Cfg.CloudflareAccountID, existing.ID, opts.Domain, opts.Domain); err != nil {
				ui.Failf("cloudflare access for %s: %v", opts.Domain, err)
				mark(5, ui.StatusFailed, fmt.Sprintf("cloudflare access for %s", opts.Domain))
				return err
			}
			appID = existing.ID
		}
		if err = cf.AttachAccessPolicy(ctx, opts.Cfg.CloudflareAccountID, appID, opts.Cfg.ZeroTrustPolicyID); err != nil {
			ui.Failf("cloudflare access for %s: %v", opts.Domain, err)
			mark(5, ui.StatusFailed, fmt.Sprintf("cloudflare access for %s", opts.Domain))
			return err
		}
		ui.OK(fmt.Sprintf("Access app for %s protected by policy %s", opts.Domain, opts.Cfg.ZeroTrustPolicyID))
		mark(5, ui.StatusOK, fmt.Sprintf("policy %s", opts.Cfg.ZeroTrustPolicyID))
	}

	// Phase 7: Record state
	ui.Phase(7, phaseNames[6])
	svc := state.Service{
		Repo:       opts.Repo,
		Slug:       s,
		ClonePath:  opts.Cfg.ClonePath,
		DeployedAt: time.Now().UTC(),
	}
	if err = state.Upsert(ctx, client, opts.Cfg.ClonePath, opts.Domain, svc); err != nil {
		ui.Failf("record state for %s: %v", opts.Domain, err)
		mark(6, ui.StatusFailed, fmt.Sprintf("state for %s", opts.Domain))
		return err
	}
	ui.OK(fmt.Sprintf("recorded state for %s", opts.Domain))
	mark(6, ui.StatusOK, "")

	completed = true
	return nil
}

// buildCloneCmd produces the remote shell command that ensures the parent dir
// exists and clones repo (with token-injected HTTPS URL) into remotePath.
// repo accepts "https://github.com/owner/name", "github.com/owner/name", or "owner/name".
func buildCloneCmd(repo, token, remotePath, clonePath string) (string, error) {
	if repo == "" {
		return "", fmt.Errorf("repo is empty")
	}
	if token == "" {
		return "", fmt.Errorf("github token is empty")
	}
	r := strings.TrimSpace(repo)
	r = strings.TrimPrefix(r, "https://")
	r = strings.TrimPrefix(r, "http://")
	r = strings.TrimSuffix(r, ".git")
	if !strings.Contains(r, "/") {
		return "", fmt.Errorf("repo %q does not look like owner/name or host/owner/name", repo)
	}
	if !strings.Contains(r, ".") {
		// "owner/name" form — assume github.com
		r = "github.com/" + r
	}
	url := fmt.Sprintf("https://x-access-token:%s@%s", token, r)
	return fmt.Sprintf("mkdir -p %s && git clone %s %s",
		shellQuote(clonePath), shellQuote(url), shellQuote(remotePath)), nil
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
