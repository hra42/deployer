# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / test

```sh
go build ./...
go test ./...
go test ./internal/deploy -run TestBuildCloneCmd   # single test
go vet ./...
```

Release binaries are produced by `.github/workflows/release.yml` on `v*` tags for `darwin/arm64`, `linux/arm64`, `linux/amd64`. The version is injected via `-ldflags "-X main.version=<tag>"`. There is no Makefile.

## Architecture

`deployer` is a Cobra CLI with two subcommands (`setup`, `deploy`) wired in `cmd/root.go`. `main.go` is a thin entrypoint.

The pipeline lives entirely in `internal/deploy/deploy.Run`. It is a fixed 6-phase sequence:

1. SSH connect (`internal/sshx`) — uses `golang.org/x/crypto/ssh` with `InsecureIgnoreHostKey` (known limitation called out in README).
2. Sync repo — `git clone` on first run, `git pull --ff-only` thereafter. The clone URL is built by `buildCloneCmd`, which normalizes `owner/name` / `github.com/...` / full URLs and injects the GitHub token as `https://x-access-token:<token>@...`.
3. Validate — checks `Dockerfile` and `docker-compose.yml` exist on the host.
4. `docker compose build --pull && docker compose up -d --remove-orphans`, with `COMPOSE_PROJECT_NAME` set to the domain slug so multiple apps coexist on one host. `build --pull` rebuilds services with a `build:` directive against fresh base images; it's a no-op for `image:`-only services.
5. Cloudflare DNS — create or update a proxied CNAME via `internal/cloudflare`.
6. Cloudflare Zero Trust — create or update an Access app and attach a pre-existing policy.

Phases 5 and 6 **skip** (don't fail) when their respective Cloudflare credentials are absent. Every phase records a `ui.SummaryItem`; the summary is printed via a `defer` so it appears even on partial failure. The pipeline does **not** roll back — a failure in phase 5 leaves the containers from phase 4 running. This is intentional.

Key cross-cutting pieces:

- `internal/config` — loads/saves `~/.deployer.yml` (YAML). `setup` writes it interactively; `deploy` reads it. CLI flags override file values.
- `internal/slug.Domain` — converts `app.example.com` → `app-example-com`. Used as both the on-host directory name (`<clone_path>/<slug>`) and the Compose project name. Changing this function will break re-deploys of existing apps.
- `internal/ui` — all stdout formatting (`Phase`, `OK`, `Warn`, `Failf`, `Summary`). Don't `fmt.Println` from pipeline code; route through `ui` so the summary stays consistent.
- `internal/cloudflare` — hand-rolled v4 REST client (no SDK). Every call goes through `Client.do`, which unwraps the standard `{success, errors, result}` envelope.
- All shell commands sent over SSH go through `shellQuote` in `deploy.go`. Anything new that interpolates user/config values into a remote command must use it.

## Conventions specific to this repo

- The host is assumed to already have Docker + Compose v2 (`docker compose`, not `docker-compose`), a running Traefik, and an external Docker network named in `traefik_network`. deployer never installs or manages these.
- Cloudflare Zero Trust policies are referenced by ID only — deployer attaches an existing policy, it does not create them.
- `dev-plan/plan.md` and `dev-plan/worker.js` exist for the install-script hosting (`deployer.hra42.lol/install`); they are not part of the CLI build.
