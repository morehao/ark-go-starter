[English](./README.md) | [简体中文](./README.zh.md)

# Quick Start

Get the full stack running from a fresh clone in **one command** (dev only):

```bash
# With Docker (recommended): auto-starts PostgreSQL + Redis, creates DB/tables,
# initializes IAM, then launches backend (:8099) and frontend (:3000).
make quickstart

# Without Docker: reuse your local PostgreSQL(5432) / Redis(6379).
make quickstart-local
```

Then open **http://localhost:3000** — you'll be redirected to the login page; log in with the
credentials printed by the command (dev default: `admin` / `Adm1n@Strong!Pass`, overridable via
`IAM_ADMIN_USER` / `IAM_ADMIN_PASS`). The command is idempotent and safe to re-run.

Verify the whole flow (login → product CRUD → dict → file upload → async → cron) with:

```bash
make quickstart-verify IAM_ADMIN_USER=admin IAM_ADMIN_PASS='<your password>'
```

> Full details (operation checklist, troubleshooting, manual steps, security notes):
> [docs/agents/quickstart.md](./docs/agents/quickstart.md).

---

# Project Overview

`ark-go-starter` is a full-stack Go Web engineering practice project. The backend is based on [Gin](https://github.com/gin-gonic/gin) + GORM (Go workspace multi-module), and the frontend is based on React + Vite + Ant Design (pnpm monorepo). It provides a layered, maintainable, and scalable structure with frontend (React) and backend (Go) separated.

---

# Features

* **Clear Project Structure**: Inspired by [project-layout](https://github.com/golang-standards/project-layout), follows layered architecture principles. Monorepo layout: `apps/` (applications: `apps/api` Go backend, `apps/web` React frontend), `libs/` (`libs/go` shared Go layer, `libs/ts` shared frontend packages), `tests/e2e`, `tools/`, `infra/`, organized for team collaboration and long-term maintenance.
* **Common Component Integration**: Includes built-in examples for PostgreSQL and Redis (Elasticsearch integration has been removed — PostgreSQL is the sole target DB).
* **Full Link Logging**: Provides a custom logging package `glog` based on `zap`, supporting full trace ID propagation across PostgreSQL, Redis, and HTTP calls.
* **Code Generation Tool**: Comes with a command-line tool `gocli` that can generate standardized code (including model, dao, object, dto, code, service, controller, router layers) based on config.
* **Swagger API Documentation**: Automatically generate interactive API docs using `swaggo` for easier frontend-backend collaboration and testing.
* **Frontend Monorepo**: Shared `libs/ts/` (tsconfig/types/api/ui) via pnpm workspace; the business app lives in `apps/web`.
* **Docker Support**: Includes a basic `Dockerfile` for containerized deployment.
* **Makefile Toolchain**: Provides a rich set of make commands to simplify code build, run, generation, Swagger docs, Docker deployment, and frontend startup.
* **Growing Golib Library**: Common utility components are abstracted and reusable via the [golib](https://github.com/morehao/golib) package.
* **Optional Account Auth (ark-iam, three modes)**: Choose via `iam.mode` — `disabled` (no auth), `op` (embed an OIDC Provider in this process — this project *is* the IAM), or `rp` (delegate to an external ark-iam, local verify only). **The shipped dev config (`apps/api/config/config.yaml`) uses `op`, so login is enforced** (quickstart prints the admin credentials); the shipped prod config uses `disabled`, and an empty `mode` also falls back to `disabled`. See [docs/agents/iam.md](./docs/agents/iam.md).

---

# Project Structure

Follows [project-layout](https://github.com/golang-standards/project-layout). Current structure:

```bash
.
├── apps
│   ├── api                     # Go backend app (Gin, :8099; logical name demo → dir api)
│   │   ├── cmd                 # Entry point
│   │   ├── client              # External clients
│   │   ├── config              # Config
│   │   ├── dao                 # Data access layer
│   │   ├── model               # Data models
│   │   ├── docs                # Swagger docs
│   │   ├── e2e                 # Backend e2e (real port)
│   │   ├── internal            # controller / dto / router / service / middleware
│   │   └── object              # Base objects
│   └── web                     # React frontend app (Vite + React + AntD, :3000)
├── libs
│   ├── go                      # Shared Go layer (code / dbclient / testsetup / token / ginhelper / iamlink / operator)
│   └── ts                      # Shared frontend packages (api / types / ui / tsconfig)
├── tests
│   └── e2e                     # E2E tests (Playwright specs + A4 golden)
├── tools                       # Scripts & SQL (quickstart.sh, sql/table_init.sql)
├── infra                       # Container & orchestration (docker/api.Dockerfile, compose/dev.yml)
├── contracts                   # Contract plane (OpenAPI 3.1 spec + verify.sh)
├── docs                        # Documentation (agents/ conventions, design-language.md)
├── go.work                     # Root Go workspace
├── package.json                # Root frontend workspace manifest
├── pnpm-workspace.yaml         # Root pnpm workspace definition
├── Makefile                    # Root Makefile (backend + frontend commands)
└── AGENTS.md                   # AI agent entry (conventions index)
```

---

# Core Features

## Code Generation

Install the CLI tool:

```bash
go install github.com/morehao/gocli@latest
```

Ensure a `code_gen.yaml` config file exists under the application directory, e.g., `apps/api/config/code_gen.yaml`.

Run code generation commands:

```bash
# Generate full module based on table
make codegen APP=demo COMMAND=module

# Generate only model code
make codegen APP=demo COMMAND=model

# Generate API endpoint code
make codegen APP=demo COMMAND=api
```

See [generate](https://github.com/morehao/gocli?tab=readme-ov-file#generate) for full documentation.

---

## API Documentation

Install Swagger tool:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Generate Swagger docs:

```bash
make swag APP=demo
```

Access docs at (dev mode):

```
http://localhost:8099/demo/redocs
```

---

## Frontend Development

Install dependencies and start the dev server from the **repo root** (pnpm workspace root):

```bash
pnpm install
pnpm dev        # default :3000, proxies /v1 → http://localhost:8099
```

Or use `make dev-frontend` to start and `make stop-frontend` to stop.
> The frontend depends on the backend; start the backend first via `make run APP=demo`.

---

## Project Deployment

Build Docker image:

```bash
make docker-build APP=demo
```

Run container:

```bash
make docker-run APP=demo
```

---

## Quickly Scaffold a New Project

Install the `cutter` tool:

```bash
go install github.com/morehao/gocli@latest
```

Run under **the root of the template project (e.g., `./`)**:

```bash
gocli cutter -d /goProject/yourAppName
```

This will scaffold a new project named `yourAppName` under `/goProject` based on the current template.

See [cutter](https://github.com/morehao/gocli) for more usage details.

---

## Related Libraries

All related components are implemented in the [golib](https://github.com/morehao/golib) package.
