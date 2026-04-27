# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

HelixFlow is an AI inference platform exposing an OpenAI-compatible API over 300+ models. The bulk of developer-facing documentation lives in `AGENTS.md` — consult it for the full command catalog, env var reference, and troubleshooting playbook. This file highlights the load-bearing context.

## Architecture

Four independent Go services talk over gRPC, fronted by HTTPS:

```
Client ──HTTPS:8443──► api-gateway ──gRPC:8081──► auth-service ──► SQLite/PostgreSQL
                           │
                           ├──gRPC:50051──► inference-pool
                           │
                           └──gRPC:8083───► monitoring
```

Each service is its own Go module under `<service>/src/` (e.g. `api-gateway/src/`, `auth-service/src/`). They depend on two local modules via `replace` directives:

- `internal/database/` — database abstraction (SQLite default, PostgreSQL for prod)
- `helixflow/{auth,inference,monitoring}/` — generated gRPC/protobuf clients and servers

Because each service is a separate module, `go build` must be run from inside that service's `src/` directory — there is no root `go.mod`. The Python SDK (`sdks/python/`) and the pytest suite (`tests/`, `test_*.py`) share a single environment pinned in `requirements-master.txt`.

`api-gateway/src/main.go` vs `main_grpc.go` / `main.py` — only the Go `main.go` + `main_grpc.go` pair is the live API gateway; `main.py` is legacy. Same for `auth-service/src/main.go` vs `main.py`.

## Common commands

Build a service (repeat per service):
```bash
cd api-gateway/src && go build -o ../bin/api-gateway .
cd auth-service/src && go build -o ../bin/auth-service .
cd inference-pool/src && go build -o ../bin/inference-pool .
cd monitoring/src && go build -o ../bin/monitoring .
```

Start everything locally: `./start_all_services.sh` (PIDs written to `logs/service_pids.txt`; kill with `kill $(cat logs/service_pids.txt)`). For manual startup with required env vars, see `AGENTS.md` → "Manual Service Startup".

Run tests:
```bash
./test_integration.sh                                  # full integration run
python -m pytest tests/integration/                    # one category
python -m pytest tests/integration/test_auth.py::test_login -v   # one test
export PYTHONWARNINGS="ignore:Unverified HTTPS request"          # quiet self-signed TLS
```

Regenerate protobuf bindings after editing `proto/*.proto`:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/*.proto
```

Health checks: `curl -k https://localhost:8443/health` (gateway), `curl http://localhost:8082/health` (auth HTTP side).

## Port map

| Service          | gRPC   | HTTP/HTTPS |
|------------------|--------|------------|
| api-gateway      | —      | 8443 (TLS) |
| auth-service     | 8081   | 8082       |
| inference-pool   | 50051  | —          |
| monitoring       | 8083   | —          |
| postgres / redis | 5432 / 6379 |       |

## Conventions that bite

- **TLS cert paths are relative to each service's working directory** (e.g. `../certs/api-gateway.crt`). If you `cd` elsewhere before launching, paths break.
- **Localhost gRPC uses insecure transport** (`grpc.WithTransportCredentials(insecure.NewCredentials())`); production uses mTLS. Don't "fix" the insecure call in dev paths.
- **Tests hit self-signed certs** — Python tests use `verify=False`; don't add cert pinning in test fixtures.
- **Token revocation requires the gRPC path**, not the HTTP shortcut. If revoked tokens are still accepted, check that the gateway logged `Auth service gRPC connection established` and that `AUTH_SERVICE_GRPC=localhost:8081` is set.
- **Database switch is env-driven**: `DATABASE_TYPE=sqlite|postgres` + `DATABASE_PATH` or `DATABASE_URL`. Default dev config is SQLite at `../data/helixflow.db`.
- **Branch naming**: `001-feature-name`. No automated CI/CD enforces quality — `./scripts/quality-gates.sh` is the manual gate.

## Known flaky / expected failures

- `test_rate_limiting_integration` fails without Redis running — start Redis via `docker-compose up -d redis` or skip.

## Where to look for more

- `AGENTS.md` — exhaustive command reference, env vars, troubleshooting
- `helixflow-technical-specification.md` — full system design
- `docs/` — API documentation
- `specs/` and `.specify/` — feature specs and templates

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->

