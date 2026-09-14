[English](./README.md) | [简体中文](./README.zh.md)

# Project Overview

`ark-go-starter` is a full-stack Go Web engineering practice project. The backend is based on [Gin](https://github.com/gin-gonic/gin) + GORM (Go workspace multi-module), and the frontend is based on React + Vite + Ant Design (pnpm monorepo). It provides a layered, maintainable, and scalable structure with frontend (React) and backend (Go) separated.

---

# Features

* **Clear Project Structure**: Inspired by [project-layout](https://github.com/golang-standards/project-layout), follows layered architecture principles. Frontend and backend are separated (`backend/` + `frontend/`), organized for team collaboration and long-term maintenance.
* **Common Component Integration**: Includes built-in examples for MySQL, Redis, and Elasticsearch.
* **Full Link Logging**: Provides a custom logging package `glog` based on `zap`, supporting full trace ID propagation across MySQL, Redis, ES, and HTTP calls.
* **Code Generation Tool**: Comes with a command-line tool `gocli` that can generate standardized code (including model, dao, object, dto, code, service, controller, router layers) based on config.
* **Swagger API Documentation**: Automatically generate interactive API docs using `swaggo` for easier frontend-backend collaboration and testing.
* **Frontend Monorepo**: Shared `packages/` (tsconfig/types/api) via pnpm workspace; business apps live in `apps/`.
* **Docker Support**: Includes a basic `Dockerfile` for containerized deployment.
* **Makefile Toolchain**: Provides a rich set of make commands to simplify code build, run, generation, Swagger docs, Docker deployment, and frontend startup.
* **Growing Golib Library**: Common utility components are abstracted and reusable via the [golib](https://github.com/morehao/golib) package.

---

# Project Structure

Follows [project-layout](https://github.com/golang-standards/project-layout). Current structure:

```bash
.
├── backend                 # Go backend project (go.work multi-module)
│   ├── apps
│   │   └── demo            # Demo example app
│   │       ├── cmd         # Entry point
│   │       ├── client      # External clients
│   │       ├── config      # Config
│   │       ├── dao         # Data access layer
│   │       ├── model       # Data models
│   │       ├── docs        # Swagger docs
│   │       ├── internal    # controller / dto / router / service / middleware
│   │       ├── object      # Base objects
│   │       └── scripts     # Dockerfile etc.
│   ├── pkg                 # Shared packages (code / dbclient / testsetup / token)
│   ├── go.work             # Go workspace file
│   └── output              # Build output
├── frontend                # React frontend project (pnpm monorepo)
│   ├── apps
│   │   └── demo-web        # Demo frontend app (Vite + React + AntD)
│   └── packages            # Shared packages (tsconfig / types / api)
├── Makefile                # Root Makefile (backend + frontend commands)
├── AGENTS.md               # Development conventions
└── docs                    # Documentation
```

---

# Core Features

## Code Generation

Install the CLI tool:

```bash
go install github.com/morehao/gocli@latest
```

Ensure a `code_gen.yaml` config file exists under the application directory, e.g., `ark-go-starter/apps/demo/config/code_gen.yaml`.

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

Install dependencies and start dev server under `frontend/`:

```bash
cd frontend
pnpm install
pnpm dev        # default :3000, proxies /v1 → http://localhost:8099
```

Or from the repo root: `make dev-frontend` to start, `make stop-frontend` to stop.
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
