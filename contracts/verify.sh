#!/usr/bin/env bash
# ============================================================================
# contracts-verify：契约生成物一致性校验（P5 试点）
#
# 校验三件事（任一失败即 exit 非 0，阻断 CI）：
#   1) spec 合法性：@redocly/cli lint contracts/openapi/openapi.json（0 error 即过）
#   2) Go client 一致：从 spec 派生 3.0.3 → oapi-codegen 重新生成到临时文件 → 与 pkg/client/client.gen.go diff
#   3) TS schema 一致：openapi-typescript 从 spec 重新生成到临时文件 → 与 packages/api-client/src/schema.ts diff
#
# 工具版本（pin，保证可复现）：
#   - @redocly/cli        2.77.0   （npx --yes @redocly/cli@2.77.0）
#   - oapi-codegen        v2.4.1   （go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1）
#   - openapi-typescript  v7.13.0  （来自 packages/api-client/node_modules，pnpm install 后提供；缺失则 npx 兜底）
#
# 环境变量（可选，未设则用默认）：
#   GOCACHE / GOMODCACHE   可指向 .dsh/tmp 隔离（沙箱下避免写 ~/.gvm）；CI 用默认即可
#   npm_config_cache       npx 缓存目录（沙箱需可写）
#   GOBIN                  oapi-codegen 安装位置
#
# 用法：make contracts-verify  或  bash contracts/verify.sh
# ============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

SPEC="contracts/openapi/openapi.json"
REDOCLY_CONFIG="contracts/openapi/redocly.yaml"
GO_GEN="pkg/client/client.gen.go"
GO_GEN_CONFIG="pkg/client/oapi-codegen.yaml"
TS_SCHEMA="packages/api-client/src/schema.ts"

REDOCLY_VERSION="2.77.0"
OAPI_CODEGEN_VERSION="v2.4.1"
OPENAPI_TS_VERSION="7.13.0"

TMP="$(mktemp -d "${TMPDIR:-/tmp}/contracts-verify.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

say()  { printf '\033[1;36m[contracts-verify]\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m[  ok  ]\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m[fail ]\033[0m %s\n' "$*" >&2; exit 1; }

[ -f "$SPEC" ]        || die "找不到 spec：$SPEC"
[ -f "$GO_GEN" ]      || die "找不到生成物：$GO_GEN"
[ -f "$TS_SCHEMA" ]   || die "找不到生成物：$TS_SCHEMA"

# ---- 1) spec 合法性（Redocly，必做）---------------------------------------
say "1/3 校验 spec 合法性（Redocly ${REDOCLY_VERSION}）…"
npx --yes "@redocly/cli@${REDOCLY_VERSION}" lint --config "$REDOCLY_CONFIG" "$SPEC" \
  || die "Redocly lint 未通过（spec 存在 error）"
ok "spec 合法（Redocly 0 error）"

# ---- 2) Go client 一致性（oapi-codegen）-----------------------------------
say "2/3 校验 Go client 生成物一致（oapi-codegen ${OAPI_CODEGEN_VERSION}）…"
if ! command -v jq >/dev/null 2>&1; then die "需要 jq（spec 3.1→3.0.3 派生）"; fi

# 解析 oapi-codegen：优先 PATH → GOBIN → $(go env GOPATH)/bin，都没有则按 pin 版本 go install
OAPI_CODEGEN="$(command -v oapi-codegen 2>/dev/null || true)"
if [ -z "$OAPI_CODEGEN" ] && [ -n "${GOBIN:-}" ] && [ -x "$GOBIN/oapi-codegen" ]; then
  OAPI_CODEGEN="$GOBIN/oapi-codegen"
fi
if [ -z "$OAPI_CODEGEN" ]; then
  _gpbin="$(go env GOPATH 2>/dev/null)/bin/oapi-codegen"
  [ -x "$_gpbin" ] && OAPI_CODEGEN="$_gpbin"
fi
if [ -z "$OAPI_CODEGEN" ]; then
  INSTALL_GOBIN="${GOBIN:-$(go env GOPATH)/bin}"
  say "  oapi-codegen 未安装，安装 ${OAPI_CODEGEN_VERSION} 到 $INSTALL_GOBIN …"
  GOBIN="$INSTALL_GOBIN" go install "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}" \
    || die "oapi-codegen 安装失败（需网络；可预装：go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}）"
  OAPI_CODEGEN="$INSTALL_GOBIN/oapi-codegen"
fi
say "  使用 oapi-codegen: $OAPI_CODEGEN"

# 派生 3.0.3（oapi-codegen 对 3.1 支持不完整）
jq '.openapi="3.0.3"' "$SPEC" > "$TMP/spec-3.0.json"
# 用临时 config 把 output 指到临时文件（oapi-codegen 会忽略 -o，必须改 config 的 output）
sed "s#^output: .*#output: $TMP/client.gen.go#" "$GO_GEN_CONFIG" > "$TMP/oapi-codegen.tmp.yaml"
"$OAPI_CODEGEN" --config "$TMP/oapi-codegen.tmp.yaml" "$TMP/spec-3.0.json" >/dev/null 2>&1 \
  || die "oapi-codegen 生成失败"
if ! diff -u "$GO_GEN" "$TMP/client.gen.go"; then
  die "Go client 生成物与 spec 不一致：请重跑生成并提交（见 pkg/client/oapi-codegen.yaml 用法）"
fi
ok "Go client 一致（${GO_GEN}）"

# ---- 3) TS schema 一致性（openapi-typescript）-----------------------------
say "3/3 校验 TS schema 生成物一致（openapi-typescript ${OPENAPI_TS_VERSION}）…"
OTS_BIN="packages/api-client/node_modules/.bin/openapi-typescript"
if [ -x "$OTS_BIN" ]; then
  OTS_CMD=("$OTS_BIN")
else
  say "  本地无 openapi-typescript，用 npx @${OPENAPI_TS_VERSION} 兜底…"
  OTS_CMD=(npx --yes "openapi-typescript@${OPENAPI_TS_VERSION}")
fi
"${OTS_CMD[@]}" "$SPEC" -o "$TMP/schema.gen.ts" >/dev/null 2>&1 \
  || die "openapi-typescript 生成失败"
if ! diff -u "$TS_SCHEMA" "$TMP/schema.gen.ts"; then
  die "TS schema 生成物与 spec 不一致：请重跑生成并提交（pnpm -C packages/api-client run generate）"
fi
ok "TS schema 一致（${TS_SCHEMA}）"

say "全部通过 ✅  spec 合法 + Go/TS 生成物均与 spec 一致"
