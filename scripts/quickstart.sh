#!/usr/bin/env bash
# ============================================================================
# ark-go-starter 一键跑通脚本（仅 dev）
#
# 目标：clone 之后一条命令，把「依赖 → 建库建表 → IAM 初始化 → 起前后端」全部搞定，
# 浏览器打开 http://localhost:3000 即可走完「登录 → 商品 CRUD → 字典 → 文件上传 →
# 异步 → 定时任务」全流程。全程幂等，重复执行不报错、不重置已初始化的数据。
#
# 用法：
#   make quickstart          # 有 Docker：自动起 PG/Redis（端口被占则自动降级复用外部）
#   make quickstart-local    # 无 Docker：复用本机已在跑的 PG/Redis（= SKIP_DEPS=1）
#   bash scripts/quickstart.sh
#
# 可用环境变量（都有合理缺省，通常无需设置）：
#   SKIP_DEPS=1            跳过依赖编排（不起容器），直接复用 127.0.0.1:5432/6379
#   PG_CONTAINER=<name>    指定用于执行 SQL 的 PG 容器名（默认自动探测）
#   IAM_ADMIN_USER         管理员用户名（默认 admin）
#   IAM_ADMIN_PASS         管理员密码（默认见下，仅 dev；支持任意强口令）
#   IAM_ADMIN_EMAIL        管理员邮箱（默认 admin@example.com，与 phone 至少一项）
#   BOOTSTRAP_TOKEN        引导令牌（默认取 config.yaml 的 install.bootstrapToken）
#   FRONTEND_PORT/HOST     打印用（默认 3000 / localhost）
#   DB_USER/DB_PASS/DB_NAME  连接本机 PG 执行建库建表用的账号（默认 postgres / 123456 / demo）。
#                          走 quickstart-local 复用本机 PG 时，若密码不是 123456，用这三个变量覆盖即可：
#                          脚本会自动生成一份改写 DSN 的配置并通过 APP_CONFIG_PATH 注入后端，无需手改 config.yaml。
#                          （若你已自行指定 APP_CONFIG_PATH，则原样透传、不生成覆盖。）
#
# 安全边界：本脚本只在 dev 使用。BOOTSTRAP_TOKEN 与管理员口令均来自环境变量或 dev 缺省，
# 不写死进 config；config 里的 dev-bootstrap-token 仅为本地便捷，生产必须覆盖。
# ============================================================================
set -euo pipefail

# ---- 路径与常量 ------------------------------------------------------------
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

APP="${APP:-demo}"
# 应用逻辑标识(APP) 与目录名(APPDIR) 解耦（D11）：demo 的 Go 目录已迁为 apps/api。
APPDIR="${APPDIR:-$( [ "$APP" = demo ] && echo api || echo "$APP" )}"
PG_HOST="127.0.0.1"
PG_PORT="5432"
REDIS_PORT="6379"
DB_NAME="${DB_NAME:-demo}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-123456}"
BACKEND_PORT="8099"
FRONTEND_HOST="${FRONTEND_HOST:-localhost}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"
API_BASE="http://127.0.0.1:${BACKEND_PORT}"
TABLE_INIT_SQL="$REPO_ROOT/apps/api/db/table_init.sql"
COMPOSE_FILE="$REPO_ROOT/deploy/compose/dev.yml"
COMPOSE_PROJECT_TAG="ark-go-starter"

# 管理员账号：dev 缺省口令用「强口令示例」，可被环境变量覆盖（绝不写进 config/README 当内置默认）
IAM_ADMIN_USER="${IAM_ADMIN_USER:-admin}"
IAM_ADMIN_PASS="${IAM_ADMIN_PASS:-admin123456}"
IAM_ADMIN_EMAIL="${IAM_ADMIN_EMAIL:-admin@example.com}"
IAM_ADMIN_NAME="${IAM_ADMIN_NAME:-平台管理员}"
IAM_TENANT_NAME="${IAM_TENANT_NAME:-Acme}"

# 引导令牌：优先环境变量；否则读 config.yaml 的 install.bootstrapToken；再不行用 dev 缺省
BOOTSTRAP_TOKEN="${BOOTSTRAP_TOKEN:-}"
if [ -z "$BOOTSTRAP_TOKEN" ]; then
  BOOTSTRAP_TOKEN="$(grep -E '^[[:space:]]*bootstrapToken:' "apps/${APPDIR}/config/config.yaml" 2>/dev/null \
    | head -1 | sed -E 's/^[^:]*:[[:space:]]*"?([^"#]*)"?.*/\1/' | xargs)"
fi
BOOTSTRAP_TOKEN="${BOOTSTRAP_TOKEN:-dev-bootstrap-token}"

PG_CONTAINER="${PG_CONTAINER:-}"

# ---- 输出工具 --------------------------------------------------------------
say()  { printf '\033[1;36m[quickstart]\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m[  ok  ]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn ]\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m[fail ]\033[0m %s\n' "$*" >&2; exit 1; }

# ---- 端口/连接探测 ---------------------------------------------------------
port_open() { nc -z -w 2 "$1" "$2" >/dev/null 2>&1; }

# wait_for_port host port 超时秒 描述
wait_for_port() {
  local host="$1" port="$2" timeout="${3:-60}" desc="${4:-$1:$2}"
  local elapsed=0
  while ! port_open "$host" "$port"; do
    if [ "$elapsed" -ge "$timeout" ]; then
      die "等待 ${desc} 就绪超时（${timeout}s）：${host}:${port} 仍不可达"
    fi
    sleep 1; elapsed=$((elapsed + 1))
  done
}

# ---- 依赖编排（PG / Redis）------------------------------------------------
ensure_deps() {
  local pg_up=0 redis_up=0
  port_open "$PG_HOST" "$PG_PORT" && pg_up=1
  port_open "$PG_HOST" "$REDIS_PORT" && redis_up=1

  if [ "$pg_up" = 1 ] && [ "$redis_up" = 1 ]; then
    ok "检测到 PG(${PG_PORT}) 与 Redis(${REDIS_PORT}) 已就绪，跳过依赖编排（复用现有实例）"
    return 0
  fi

  if [ "${SKIP_DEPS:-0}" = "1" ]; then
    [ "$pg_up" = 1 ]  || die "SKIP_DEPS=1 但 PG(${PG_PORT}) 未就绪，请先启动本机 PostgreSQL"
    [ "$redis_up" = 1 ] || die "SKIP_DEPS=1 但 Redis(${REDIS_PORT}) 未就绪，请先启动本机 Redis"
    return 0
  fi

  command -v docker >/dev/null 2>&1 || die "未检测到 docker，且 PG/Redis 未就绪。请安装 Docker，或用 make quickstart-local 复用本机 PG/Redis"
  docker compose version >/dev/null 2>&1 || die "docker compose(v2) 不可用，请升级 Docker，或用 make quickstart-local"

  say "启动依赖容器（PG:${PG_PORT} / Redis:${REDIS_PORT}）…"
  # 只补起缺的（compose up 会按需创建，已有容器不动）
  docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT_TAG" up -d

  say "等待 PG / Redis 就绪…"
  [ "$pg_up" = 1 ]    || wait_for_port "$PG_HOST" "$PG_PORT" 90 "PostgreSQL"
  [ "$redis_up" = 1 ] || wait_for_port "$PG_HOST" "$REDIS_PORT" 90 "Redis"
  ok "依赖容器已就绪"
}

# ---- 解析用于执行 SQL 的 PG 容器 ------------------------------------------
# 无宿主 psql 时用 docker exec；自动探测 compose 起的容器 -> 任意 postgres 容器。
resolve_pg_container() {
  [ -n "$PG_CONTAINER" ] && return 0
  local candidates="ark-go-starter-postgres postgres18 postgres16 postgres"
  local c
  for c in $candidates; do
    if docker inspect -f '{{.State.Running}}' "$c" >/dev/null 2>&1; then
      if [ "$(docker inspect -f '{{.State.Running}}' "$c" 2>/dev/null)" = "true" ]; then
        PG_CONTAINER="$c"; return 0
      fi
    fi
  done
  # 兜底：任意镜像名含 postgres 的运行中容器
  c="$(docker ps --filter 'ancestor=postgres' --format '{{.Names}}' 2>/dev/null | head -1)"
  [ -z "$c" ] && c="$(docker ps --format '{{.Names}}\t{{.Image}}' 2>/dev/null | awk -F'\t' 'tolower($2) ~ /postgres/ {print $1; exit}')"
  [ -n "$c" ] && PG_CONTAINER="$c"
  return 0
}

# psql 执行（优先 docker exec PG 容器；无容器则退回宿主 psql）
psql_exec() {
  resolve_pg_container
  if [ -n "$PG_CONTAINER" ]; then
    docker exec -i -e PGPASSWORD="$DB_PASS" "$PG_CONTAINER" \
      psql -U "$DB_USER" -h 127.0.0.1 -v ON_ERROR_STOP=1 "$@"
  elif command -v psql >/dev/null 2>&1; then
    PGPASSWORD="$DB_PASS" psql -U "$DB_USER" -h "$PG_HOST" -p "$PG_PORT" -v ON_ERROR_STOP=1 "$@"
  else
    die "既没有可用的 PG 容器，也没有宿主 psql，无法执行 SQL。请装 Docker 或 psql，或设置 PG_CONTAINER"
  fi
}

# ---- 建库建表 --------------------------------------------------------------
ensure_db() {
  say "确保数据库 ${DB_NAME} 存在…"
  local exists
  exists="$(psql_exec -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" | tr -d '[:space:]')"
  if [ "$exists" = "1" ]; then
    ok "数据库 ${DB_NAME} 已存在"
  else
    psql_exec -d postgres -c "CREATE DATABASE \"${DB_NAME}\"" >/dev/null
    ok "数据库 ${DB_NAME} 已创建"
  fi
}

run_sql() {
  say "执行建表脚本 table_init.sql（CREATE TABLE IF NOT EXISTS，幂等）…"
  [ -f "$TABLE_INIT_SQL" ] || die "找不到建表脚本：$TABLE_INIT_SQL"
  psql_exec -d "$DB_NAME" -f - < "$TABLE_INIT_SQL" >/dev/null
  ok "业务表已就绪"
}

# ---- 依赖安装（fresh clone 保护）------------------------------------------
ensure_app_deps() {
  if [ ! -d "$REPO_ROOT/node_modules" ]; then
    command -v pnpm >/dev/null 2>&1 || die "未检测到 pnpm（前端需要 pnpm>=9）。请先安装：npm i -g pnpm"
    say "首次运行，安装前端依赖（pnpm install）…"
    pnpm install
    ok "前端依赖已安装"
  fi
  # go 依赖：go.work 下各模块按需下载（首次 go run 会自动拉，这里显式预热更稳）。
  # 下载失败不致命（离线/已有缓存时 go run 仍可构建），故仅 best-effort。
  say "预热 Go 依赖（go mod download）…"
  ( cd "$REPO_ROOT" && go work edit -json 2>/dev/null | grep -o '\"DiskPath\": \"[^\"]*\"' | awk '{gsub(/\"/, "", $2); print $2}' \
    | while read -r m; do ( cd "$REPO_ROOT/$m" 2>/dev/null && go mod download >/dev/null 2>&1 ) || true; done ) || true
}

# ---- 生成配置覆盖（DB 账号与 config.yaml 对齐）---------------------------
# config.yaml 的 DSN 写死为 postgres:123456@.../demo；若用户用 DB_USER/DB_PASS/DB_NAME 覆盖，
# 应用仍会按 config.yaml 连库导致不一致。这里在需要时生成一份覆盖 config，通过 APP_CONFIG_PATH 注入。
# 仅在 (DB_USER,DB_PASS,DB_NAME) != (postgres,123456,demo) 且用户未自行指定 APP_CONFIG_PATH 时生成。
prepare_config_override() {
  if [ -n "${APP_CONFIG_PATH:-}" ]; then
    say "检测到外部 APP_CONFIG_PATH，跳过 DB 账号覆盖生成（复用用户配置）"
    return 0
  fi
  if [ "$DB_USER" = "postgres" ] && [ "$DB_PASS" = "123456" ] && [ "$DB_NAME" = "demo" ]; then
    return 0  # 与 config.yaml 默认一致，无需覆盖
  fi
  local base="$REPO_ROOT/apps/${APPDIR}/config/config.yaml"
  [ -f "$base" ] || die "找不到基础配置：$base"
  local out="$REPO_ROOT/.local/log/quickstart-config.${APP}.yaml"
  mkdir -p "$REPO_ROOT/.local/log"
  # 仅替换 db_configs 里那条 postgres DSN 的 userinfo@host:port/db 段（保留 ?sslmode=... 等参数）。
  # DSN 形如：postgres://postgres:123456@127.0.0.1:5432/demo?sslmode=...&TimeZone=...
  local dsn="postgres://${DB_USER}:${DB_PASS}@${PG_HOST}:${PG_PORT}/${DB_NAME}"
  local host_re="${PG_HOST//./\\.}"
  sed -E "s#postgres://[^/@]*@${host_re}:${PG_PORT}/[^\"?]+#${dsn}#" "$base" > "$out"
  APP_CONFIG_PATH="$out"
  say "已生成 DB 账号覆盖配置：$out（user=$DB_USER db=$DB_NAME）"
}

# ---- 起后端（后台）并等待 /iam/install/status ----------------------------
start_backend() {
  say "启动后端（8099）…"
  mkdir -p "$REPO_ROOT/.local/log"
  # 复用 make dev-all 的清理，避免端口残留
  make -s stop-dev >/dev/null 2>&1 || true
  # 后台起后端；BOOTSTRAP_TOKEN 通过环境变量注入（env 优先于 config 文件）。
  # APP_CONFIG_PATH 若外部已指定则透传（便于隔离测试/自定义配置），否则用默认 config.yaml。
  (
    cd "apps/${APPDIR}/cmd"
    APP_CONFIG_PATH="${APP_CONFIG_PATH:-}" BOOTSTRAP_TOKEN="$BOOTSTRAP_TOKEN" go run . > "$REPO_ROOT/.local/log/quickstart-backend.log" 2>&1 &
    echo $! > "$REPO_ROOT/.local/log/quickstart-backend.pid"
  )
  local elapsed=0
  until curl -sf "${API_BASE}/iam/install/status" >/dev/null 2>&1; do
    if [ "$elapsed" -ge 120 ]; then
      warn "后端 8099 未在 120s 内就绪，最后 40 行日志："
      tail -n 40 "$REPO_ROOT/.local/log/quickstart-backend.log" >&2 || true
      die "后端启动失败"
    fi
    sleep 1; elapsed=$((elapsed + 1))
  done
  ok "后端已就绪：${API_BASE}（日志 .local/log/quickstart-backend.log）"
}

# ---- IAM 初始化（幂等）----------------------------------------------------
init_iam() {
  say "检查 IAM 初始化状态…"
  local status_json initialized token_required schema_ready
  status_json="$(curl -sf "${API_BASE}/iam/install/status")"
  # 极简解析（不依赖 jq，且兼容 macOS/BSD 与 GNU 工具链）：grep -o 提取 "key":value 再截值。
  json_bool() { printf '%s' "$1" | grep -o "\"$2\":[a-z]*" | head -1 | sed 's/.*://'; }
  initialized="$(json_bool "$status_json" initialized)"
  token_required="$(json_bool "$status_json" tokenRequired)"
  schema_ready="$(json_bool "$status_json" schemaReady)"

  if [ "$schema_ready" = "false" ]; then
    die "/iam/install/status 报告 schemaReady=false，表结构未就绪（请确认 table_init.sql 已执行）"
  fi

  if [ "$initialized" = "true" ]; then
    ok "IAM 已初始化，跳过（不重置既有账号与数据）"
    return 0
  fi

  say "首次初始化 IAM（创建管理员 ${IAM_ADMIN_USER}）…"
  if [ "$token_required" = "true" ] && [ -z "$BOOTSTRAP_TOKEN" ]; then
    die "服务端要求引导令牌，但未提供 BOOTSTRAP_TOKEN"
  fi
  local body resp code
  body="$(printf '{"adminUsername":"%s","adminPassword":"%s","adminEmail":"%s","adminName":"%s","tenantName":"%s"}' \
    "$IAM_ADMIN_USER" "$IAM_ADMIN_PASS" "$IAM_ADMIN_EMAIL" "$IAM_ADMIN_NAME" "$IAM_TENANT_NAME")"
  resp="$(curl -s -w '\n%{http_code}' -X POST "${API_BASE}/iam/install/initialize" \
    -H 'Content-Type: application/json' \
    -H "X-Bootstrap-Token: ${BOOTSTRAP_TOKEN}" \
    -d "$body")"
  code="$(printf '%s' "$resp" | tail -n1)"
  if [ "$code" = "200" ]; then
    ok "IAM 初始化成功（管理员：${IAM_ADMIN_USER}）"
    return 0
  fi
  if [ "$code" = "409" ]; then
    # 409 = 系统已完成初始化（并发/竞态或状态查询竞态）：视为幂等成功，不报错。
    ok "IAM 已完成初始化（409，跳过）"
    return 0
  fi
  warn "初始化响应（HTTP ${code}）：$(printf '%s' "$resp" | sed '$d')"
  die "IAM 初始化失败（HTTP ${code}）。请检查 BOOTSTRAP_TOKEN 是否与后端一致"
}

# ---- 起前端 ----------------------------------------------------------------
start_frontend() {
  say "启动前端（${FRONTEND_HOST}:${FRONTEND_PORT}）…"
  # 停掉可能残留的旧 vite，避免端口漂移
  pkill -f "vite" >/dev/null 2>&1 || true
  pnpm dev
}

# ---- main ------------------------------------------------------------------
main() {
  say "ark-go-starter quickstart（dev）"
  ensure_deps
  ensure_app_deps
  ensure_db
  run_sql
  prepare_config_override
  start_backend
  init_iam

  echo ""
  ok "后端与依赖已就绪，准备启动前端。"
  echo "=============================================="
  echo "  前端访问地址: http://${FRONTEND_HOST}:${FRONTEND_PORT}"
  echo "  后端接口:     ${API_BASE}"
  echo "  登录账号:     ${IAM_ADMIN_USER}"
  echo "  登录密码:     ${IAM_ADMIN_PASS}   (仅 dev 缺省，可用 IAM_ADMIN_PASS 覆盖)"
  echo "  后端日志:     .local/log/quickstart-backend.log"
  echo "  停止:         Ctrl+C（仅停前端）；后端: make stop-dev"
  echo "=============================================="
  echo ""
  start_frontend
}

main "$@"
