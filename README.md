# deployer

A small Go CLI that ships a containerised app to a single host with one command. Point it at a GitHub repo and a domain; it clones the repo over SSH, brings up the Compose stack behind a Traefik instance you already run, sets the DNS record on Cloudflare, and protects the app with a pre-existing Cloudflare Zero Trust policy.

The boring config (SSH host, key path, GitHub token, Cloudflare credentials) lives in `~/.deployer.yml`. You set it up once with `deployer setup` and re-use it for every deploy.

## Install

One-line install (downloads the latest release binary, verifies its SHA-256, installs to `/usr/local/bin` or `$HOME/.local/bin`):

```sh
curl -fsSL https://deployer.hra42.lol/install | sh
```

Pin a specific version or override the install location:

```sh
curl -fsSL https://deployer.hra42.lol/install | VERSION=v0.1.0 sh
curl -fsSL https://deployer.hra42.lol/install | INSTALL_DIR=$HOME/.local/bin sh
```

Supported targets: `darwin/arm64`, `linux/arm64`, `linux/amd64`. macOS x86_64 is not built — install from source instead.

From source:

```sh
go install github.com/hra42/deployer@latest
```

## First-time setup

```sh
deployer setup
```

Walks through every config field interactively and writes `~/.deployer.yml` (overwriting any existing file). See [Config reference](#config-reference) for what each field means.

## Deploy

```sh
deployer deploy --repo owner/name --domain app.example.com
```

`--repo` accepts any of `owner/name`, `github.com/owner/name`, or `https://github.com/owner/name`.

Expected output on a successful run:

```
==> [1/6] Connect to host
    ✓ connected to deploy@1.2.3.4

==> [2/6] Sync repo
    git clone into /srv/apps/app-example-com
    ✓ repo synced

==> [3/6] Validate compose files
    ✓ Dockerfile and docker-compose.yml present

==> [4/6] Bring up containers
    ✓ project app-example-com up

==> [5/6] Cloudflare DNS
    creating CNAME app.example.com → host.example.com
    ✓ proxied CNAME app.example.com → host.example.com

==> [6/6] Cloudflare Zero Trust
    creating Access app for app.example.com
    ✓ Access app for app.example.com protected by policy <id>

── Summary ──
    ✓ Connect to host        ok — deploy@1.2.3.4
    ✓ Sync repo              ok — /srv/apps/app-example-com
    ✓ Validate compose files ok
    ✓ Bring up containers    ok — project app-example-com
    ✓ Cloudflare DNS         ok — CNAME app.example.com → host.example.com
    ✓ Cloudflare Zero Trust  ok — policy <id>
    ✓ deploy complete: app.example.com
```

If a phase fails, the summary still prints — so you can see at a glance which phases ran, which were skipped, and which failed (e.g. containers up but DNS broken).

## Host requirements

The target host must already have:

- **Docker** with the **Compose v2** plugin (`docker compose ...`, not `docker-compose`).
- A **Traefik** container running on the host, configured with HTTPS entrypoints and a cert resolver.
- An **external Docker network** that Traefik watches and that every deployed app joins.

deployer does **not** install or manage any of these. It assumes they're set up.

A minimal Traefik host setup (for reference) looks like:

```yaml
# /srv/traefik/docker-compose.yml — runs on the host, started once
services:
  traefik:
    image: traefik:v3
    restart: unless-stopped
    command:
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --providers.docker.network=web
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.le.acme.email=you@example.com
      - --certificatesresolvers.le.acme.storage=/letsencrypt/acme.json
      - --certificatesresolvers.le.acme.tlschallenge=true
    ports: ["80:80", "443:443"]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt
    networks: [web]

networks:
  web:
    external: true
```

Create the network once: `docker network create web`. Then put `web` (or whatever you named it) in `traefik_network` in your config.

## App repo requirements

Every repo you deploy through deployer **must** include:

1. A `Dockerfile` at the repo root.
2. A `docker-compose.yml` at the repo root that joins the external Traefik network and declares Traefik labels for the routed service.

Minimal copy-pasteable template:

```yaml
# docker-compose.yml in your app repo
services:
  app:
    build: .
    restart: unless-stopped
    networks: [web]
    labels:
      - traefik.enable=true
      - traefik.docker.network=web
      - traefik.http.routers.app.rule=Host(`app.example.com`)
      - traefik.http.routers.app.entrypoints=websecure
      - traefik.http.routers.app.tls.certresolver=le
      - traefik.http.services.app.loadbalancer.server.port=8080

networks:
  web:
    external: true
```

Notes:

- Replace `web` with whatever you set for `traefik_network`.
- `Host(...)` must match the `--domain` you pass to `deployer deploy`.
- `loadbalancer.server.port` should match whatever port your container listens on internally.
- At deploy time, `COMPOSE_PROJECT_NAME` is set to a slug of the domain (e.g. `app.example.com` → `app-example-com`), so multiple deploys on the same host don't collide. Keep service names generic (`app`, `web`, `api`) — they'll be namespaced by the project name.

If `Dockerfile` or `docker-compose.yml` is missing, deploy hard-fails with a clear message.

## Config reference

All fields live in `~/.deployer.yml`. CLI flags (where they exist) override the file at runtime.

| Field | Required | Description |
| --- | --- | --- |
| `ssh_host` | yes | Target host as `user@ip-or-hostname`. |
| `ssh_key_path` | yes | Path to the private SSH key for that host. |
| `github_token` | yes | GitHub PAT with `repo` scope; injected into the HTTPS clone URL. |
| `clone_path` | yes | Base directory on the host where repos are cloned (e.g. `/srv/apps`). Each app lives at `<clone_path>/<domain-slug>`. |
| `traefik_network` | yes | Name of the external Docker network Traefik watches. |
| `cloudflare_api_token` | optional | Cloudflare API token. If unset, DNS and Zero Trust phases are skipped with a warning. |
| `cloudflare_zone_id` | optional | Zone ID for the domain's parent zone. Required for the DNS phase. |
| `cloudflare_account_id` | optional | Cloudflare account ID. Required for the Zero Trust phase. |
| `zero_trust_policy_id` | optional | ID of a pre-existing Access policy to attach to each deployed app. |
| `cname_target` | optional | What every CNAME points at. Defaults to `host.example.com`. |

Example:

```yaml
ssh_host: deploy@1.2.3.4
ssh_key_path: /Users/me/.ssh/id_ed25519
github_token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
clone_path: /srv/apps
traefik_network: web
cloudflare_api_token: cf_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
cloudflare_zone_id: 0123456789abcdef0123456789abcdef
cloudflare_account_id: fedcba9876543210fedcba9876543210
zero_trust_policy_id: 11111111-2222-3333-4444-555555555555
cname_target: host.example.com
```

If the Cloudflare fields are blank, deployer still ships the containers — it just skips DNS and Zero Trust and tells you so in the summary.

## What deployer does NOT do

- Install Docker or Compose on the host.
- Set up or run Traefik — bring your own.
- Create or manage Cloudflare Zero Trust policies. It only **attaches** a policy you've already created, by ID.
- Roll back on partial failure. If phase 5 fails after phase 4 succeeded, the containers stay up; the summary tells you so. Fix the underlying issue and re-run.
- Verify SSH host keys against `known_hosts` (yet). The current build uses `InsecureIgnoreHostKey`. Don't run this against hosts you don't already trust.

## License

[The Unlicense](LICENSE) — public domain. Do whatever you want.
