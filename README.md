# CloudChamber

Go/Kubernetes service platform with a layered **api / app / business / foundation** layout. The primary API service is **CloudChamber** (`api/services/cloudchamber`), with separate **auth** and **metrics** processes.

**Module:** `github.com/MKAbdElrahman/cloudchamber`  
**Go:** 1.27.3 (see `go.mod`)

## Layout

```
api/          Service entrypoints (cloudchamber, auth, metrics) and admin tooling
app/          Application layer (HTTP routes, DTOs, middleware wiring)
business/     Domain and data access
foundation/   Reusable packages (web, logger, otel, worker, …)
zarf/         Docker, Compose, Kubernetes, Helm
```

Cross-layer conventions: Structurizr docs under `doc/architecture/c4/docs/` (`make c4-up`), agent skills in `.agents/skills/`.

**C4 / Structurizr:** `doc/architecture/c4/workspace.dsl`, supplementary docs in `doc/architecture/c4/docs/`, ADRs in `doc/architecture/c4/decisions/`. `make c4-up` / `make c4-down`, `make c4-validate`.

## Prerequisites

- Go 1.27.3+ on `PATH` (`/usr/local/go/bin` and `$GOPATH/bin`)
- Docker (tests and local deployments use a Postgres container named `servicetest` or Compose/KIND images)
- For the full Kubernetes dev flow: `kind`, `kubectl`, `kustomize` (see `make dev-gotooling` and `make dev-brew`)

## Clone

```bash
git clone git@github.com:MKAbdElrahman/cloudchamber.git
cd cloudchamber
```

Vendor is checked in; prefer `go build -mod=vendor` / `go test -mod=vendor` if you do not run `go mod download`.

## Tooling (one-time)

```bash
make dev-gotooling   # staticcheck, govulncheck, protoc plugins, …
make dev-docker      # pull base images
make dev-brew        # optional: kind, kubectl, kustomize via Homebrew
```

For Go 1.27, install a recent **staticcheck** if `make lint` fails on export data version errors:

```bash
go install honnef.co/go/tools/cmd/staticcheck@master
```

## Tests

```bash
make test
```

Integration tests start Docker Postgres (`servicetest`) and reuse it while it is running. The container is removed when it stops. `make test-down` removes it and its data.

Run only unit/integration tests without lint/vuln:

```bash
make test-only
```

## Run locally

### Kubernetes (KIND) — full stack

```bash
make dev-run
# or: make build && make dev-up && make dev-load && make dev-update-apply
```

Useful follow-ups:

```bash
make dev-status
make dev-logs
make token
export TOKEN='<paste>'
make users
make dev-down
```

### Docker Compose

```bash
make build
make compose-build-up
make compose-down    # stop
```

### Single binary on the host

With Postgres (e.g. Compose) and env vars set (`CLOUDCHAMBER_DB_*`):

```bash
export CLOUDCHAMBER_DB_HOST=localhost
make seed
make run-auth    # terminal 1
make run         # terminal 2 — cloudchamber API
make run-help    # configuration help
```

## Agent skills

The repo includes [Agent Skills](.agents/skills) for AI-assisted work (layering rules, Go style, PR review). See `AGENTS.md` for how they are intended to be loaded.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Includes code originally from the Apache 2.0 [ardanlabs/service](https://github.com/ardanlabs/service) starter (see [LICENSE](LICENSE)). No affiliation with Ardan Labs training or course materials.
