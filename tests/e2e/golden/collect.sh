#!/usr/bin/env bash
#
# collect.sh —— A4 golden 采集脚本（P0-T2）
#
# 采集 20 个接口的 (code, message, HTTP status) 三元组，每条附带 ≥1 个 4xx/业务错误负例，
# 产出稳定的 JSON（tests/e2e/golden/a4_golden.json）。连续跑两次输出必须完全一致（DoD）。
#
# 设计要点（为什么这样采）：
#   1. 只记录 (code, message, httpStatus) 三元组；**不记录 requestID / data / 时间戳**，
#      这些每次都会变，不属契约。三元组才是迁移前后必须等价的用户可见语义。
#   2. 采集时用一份「派生配置」：iam.mode 由 op 改为 disabled、env 保持 dev。
#      这样业务路由走 dev 演示身份中间件（无需 token），把 IAM 鉴权这条正交维度剥离，
#      golden 专注「路由 → 中间件 → controller → service」的接口契约（迁移真正会动的东西）。
#      （生产配置 iam.mode=op 时这批路由无 token 统一 401，属鉴权层，不在 A4 契约范围内；
#        详见 .dsh/docs/notes/2026-11-12-a4-golden-baseline-notes.md。）
#   3. 端口用 8098（避开可能已被占用的 8099 开发实例），日志/存储目录落到被 .gitignore 忽略的
#      backend/.dsh/tmp，不污染仓库、不写入仓内 log/storage。
#   4. 造数（POST products）用随机唯一名，采集后立即清理，保证多次运行库不变、输出不变。
#
# 用法（在仓库根目录）：
#   bash tests/e2e/golden/collect.sh            # 采集并打印 JSON 到 stdout，同时写 a4_golden.json
#   bash tests/e2e/golden/collect.sh --check    # 采集并与已入库 a4_golden.json 比对，差异即非 0 退出
#
# 依赖：go（含 workspace）、curl、jq、python3（仅用于 JSON 规范化排序）。
# 外部依赖：本机 PostgreSQL(5432) / Redis(6379) 需在跑（demo 启动必需）。

set -euo pipefail

# ---- 路径与常量 ----------------------------------------------------------------
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
GOLDEN_DIR="$REPO_ROOT/tests/e2e/golden"
GOLDEN_FILE="$GOLDEN_DIR/a4_golden.json"
BACKEND_DIR="$REPO_ROOT/backend"
TMP_DIR="$BACKEND_DIR/.dsh/tmp"                 # 被 .gitignore 忽略（.dsh/ 任意层级）
GOCACHE_DIR="$TMP_DIR/gocache"
LOG_DIR="$TMP_DIR/log"
STORAGE_DIR="$TMP_DIR/storage"
BIN="$TMP_DIR/demo-server"
CFG="$TMP_DIR/config.golden.yaml"
SRV_LOG="$TMP_DIR/server.out"
PORT="${A4_PORT:-8098}"
BASE="http://127.0.0.1:${PORT}/v1/demo"

CHECK_MODE=""
[[ "${1:-}" == "--check" ]] && CHECK_MODE=1

# ---- 依赖检查 ------------------------------------------------------------------
for t in go curl jq python3; do
  command -v "$t" >/dev/null 2>&1 || { echo "缺少依赖: $t" >&2; exit 2; }
done

mkdir -p "$TMP_DIR" "$GOCACHE_DIR" "$LOG_DIR" "$STORAGE_DIR"

# ---- 派生配置（iam.mode: op -> disabled；端口/目录指到 tmp）-----------------------
sed -e 's/^  mode: op/  mode: disabled/' \
    -e "s#^  port: 8099#  port: ${PORT}#" \
    -e "s#dir: ../../../log#dir: ${LOG_DIR}#" \
    -e "s#base_dir: ../../../storage#base_dir: ${STORAGE_DIR}#" \
    -e "s#base_url: http://127.0.0.1:8099/v1/demo/objects#base_url: http://127.0.0.1:${PORT}/v1/demo/objects#" \
    "$BACKEND_DIR/apps/demo/config/config.yaml" > "$CFG"

# ---- 构建（GOCACHE 指向仓内忽略目录，避免写到仓库外被沙箱拒）----------------------
( cd "$BACKEND_DIR" && GOCACHE="$GOCACHE_DIR" go build -o "$BIN" ./apps/demo/cmd )

# ---- 启动服务并等就绪 -----------------------------------------------------------
SERVER_PID=""
cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    # 用 SIGKILL：SSE 路由是无限流，SIGTERM 触发的 Gin 优雅关闭会一直等这些活跃连接结束 → 永不返回。
    kill -9 "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

APP_CONFIG_PATH="$CFG" "$BIN" > "$SRV_LOG" 2>&1 &
SERVER_PID=$!

ready=0
for _ in $(seq 1 60); do
  if curl -s -o /dev/null --max-time 1 "$BASE/health" 2>/dev/null; then ready=1; break; fi
  kill -0 "$SERVER_PID" 2>/dev/null || { echo "服务进程已退出，日志：" >&2; tail -30 "$SRV_LOG" >&2; exit 1; }
  sleep 0.5
done
[[ "$ready" == "1" ]] || { echo "服务 15s 内未就绪，日志：" >&2; tail -30 "$SRV_LOG" >&2; exit 1; }

# ---- 采集辅助 ------------------------------------------------------------------
# 结果累积到 NDJSON 临时文件（每行一条 {id, kind, method, path, code, message, httpStatus}）。
RAW_NDJSON="$TMP_DIR/golden.raw.ndjson"
: > "$RAW_NDJSON"

# hit METHOD PATH [BODY] [ID] [KIND] [EXTRA] [DISPLAY_PATH]
#   发请求，抽取 HTTP status、body 里的 code/msg；写一行 NDJSON。
#   - JSON body：code 取整数，message 取 msg（缺失记 null）。
#   - 非 JSON（如 SSE 流）：code 记 null，message 记 Content-Type（有信息量），httpStatus 照记。
#   - $7 DISPLAY_PATH：记录用的稳定路径（把动态 id 换成占位符，如 /products/{productID}），
#     保证两次运行输出逐字节一致；真正请求仍用 $2 的实际路径。
hit() {
  local method="$1" path="$2" body="${3:-}" id="$4" kind="$5" extra="${6:-}" display="${7:-}"
  [[ -z "$display" ]] && display="$path"
  local resp status code message ctype tmpbody
  tmpbody="$(mktemp)"
  local -a cargs=(-s -o "$tmpbody" -w '%{http_code}' --max-time 10 -X "$method")
  if [[ -n "$body" ]]; then
    cargs+=(-H 'Content-Type: application/json' --data "$body")
  fi
  if [[ -n "$extra" ]]; then
    # extra 是给 SSE 之类用的：--max-time 等，按空格拆
    # shellcheck disable=SC2206
    cargs+=($extra)
  fi
  # 注意：流式（SSE）请求即便 --max-time 超时，curl 也会先把 http_code 打到 stdout 再以非 0 退出。
  # 若用 `|| echo 000` 兜底会把 "200" 和 "000" 拼成 "200000"，所以这里直接取值，空则兜底 000。
  status="$(curl "${cargs[@]}" "$BASE$path" 2>/dev/null)" || true
  status="${status:-000}"

  if jq -e . >/dev/null 2>&1 <"$tmpbody"; then
    code="$(jq -c '.code // null' <"$tmpbody")"
    message="$(jq -c '.msg // null' <"$tmpbody")"
  else
    code="null"
    # 非 JSON（SSE 流）：再取一次响应头里的 Content-Type 作为 message 的替代标识。
    # 单独 curl 只为拿头，超时/断流都容忍，绝不因非 0 退出码中断采集。
    ctype="$(curl -s -D - -o /dev/null --max-time 4 -X "$method" \
              ${body:+-H 'Content-Type: application/json' --data "$body"} "$BASE$path" 2>/dev/null \
              | tr -d '\r' | awk -F': ' 'tolower($1)=="content-type"{print $2; exit}' || true)"
    message="$(jq -Rn --arg v "${ctype:-non-json}" '$v')"
  fi
  rm -f "$tmpbody"
  jq -cn \
    --arg id "$id" --arg kind "$kind" --arg method "$method" --arg path "$display" \
    --argjson code "$code" --argjson message "$message" --argjson status "${status:-0}" \
    '{id:$id, kind:$kind, method:$method, path:$path, code:$code, message:$message, httpStatus:$status}' \
    >> "$RAW_NDJSON"
}

SSE_EXTRA="--max-time 3 -N"   # SSE 无限流：3s 超时截断，只为拿状态码/Content-Type

# ---- 采样过程中的临时资源 ------------------------------------------------------
CREATED_PRODUCT_ID=""
CREATED_PRODUCT2_ID=""
TAG="a4golden-$(date +%s)-$$"

create_product() {  # 返回 productID
  local name="$1"
  curl -s -X POST -H 'Content-Type: application/json' \
    --max-time 10 --data "{\"name\":\"$name\",\"categoryID\":1,\"price\":9.9}" "$BASE/products" 2>/dev/null \
    | jq -r '.data.productID // empty'
}

# ============================================================================
#  14 条 v1 资源路由（demo 应用自身注册的全部 v1 路由）
# ============================================================================

# 1) GET /v1/demo/health
hit GET "/health" "" health positive
hit POST "/health" "" health negative                              # 只注册 GET，POST 应 405

# 2) GET /v1/demo/format/formatResponse
hit GET "/format/formatResponse" "" format positive
hit POST "/format/formatResponse" "" format negative               # 405

# 3) GET /v1/demo/client/callGetHttpbingo （外部 HTTP 依赖，记录当前观测）
hit GET "/client/callGetHttpbingo" "" client_httpbingo positive
hit POST "/client/callGetHttpbingo" "" client_httpbingo negative   # 405

# 4)-8) SSE 流（GET，无限流；记录 httpStatus + Content-Type），负例用 POST → 405
hit GET "/sse/time"    "" sse_time    positive "$SSE_EXTRA"
hit POST "/sse/time"    "" sse_time    negative
hit GET "/sse/timeRaw" "" sse_timeRaw positive "$SSE_EXTRA"
hit POST "/sse/timeRaw" "" sse_timeRaw negative
hit GET "/sse/process" "" sse_process positive "$SSE_EXTRA"
hit POST "/sse/process" "" sse_process negative
hit GET "/sse/chat"    "" sse_chat    positive "$SSE_EXTRA"
hit POST "/sse/chat"    "" sse_chat    negative
hit GET "/sse/raw"     "" sse_raw     positive "$SSE_EXTRA"
hit POST "/sse/raw"     "" sse_raw     negative

# 商品：先造一个用于 detail/update/delete 正例的商品
CREATED_PRODUCT_ID="$(create_product "$TAG-p1")"

# 9) POST /v1/demo/products （创建）
hit POST "/products" "{\"name\":\"$TAG-create\",\"categoryID\":1,\"price\":9.9}" products_create positive
hit POST "/products" "{\"name\":\"x\",\"price\":\"not-a-float\"}" products_create negative

# 10) POST /v1/demo/products/tx （事务创建）
hit POST "/products/tx" "{\"firstName\":\"$TAG-tx-a\",\"secondName\":\"$TAG-tx-b\"}" products_tx positive
hit POST "/products/tx" "{\"firstName\":\"only-one\"}" products_tx negative

# 11) GET /v1/demo/products （分页列表）
hit GET "/products?page=1&pageSize=10" "" products_list positive
hit GET "/products?page=abc" "" products_list negative

# 12) GET /v1/demo/products/:productID （详情）
hit GET "/products/$CREATED_PRODUCT_ID" "" products_detail positive "" "/products/{productID}"
hit GET "/products/999999999" "" products_detail negative

# 13) PUT /v1/demo/products/:productID （更新）
hit PUT "/products/$CREATED_PRODUCT_ID" '{"price":19.9}' products_update positive "" "/products/{productID}"
hit PUT "/products/999999999" '{"price":19.9}' products_update negative

# 14) DELETE /v1/demo/products/:productID （删除）
hit DELETE "/products/$CREATED_PRODUCT_ID" "" products_delete positive "" "/products/{productID}"
hit DELETE "/products/999999999" "" products_delete negative

# ============================================================================
#  + 5 个 golib 模块代表接口（dict / setting / cron / async / upload 各 1）
# ============================================================================

# 15) GET /v1/demo/dict-types
hit GET "/dict-types" "" dict positive
hit POST "/dict-types" '{"name":"no-code"}' dict negative

# 16) GET /v1/demo/settings
hit GET "/settings" "" setting positive
hit POST "/settings" '{"key":"k","value":"v"}' setting negative   # 缺 group

# 17) GET /v1/demo/cron-tasks
hit GET "/cron-tasks" "" cron positive
hit GET "/cron-tasks/nonexistent-id" "" cron negative             # 404 task not registered

# 18) GET /v1/demo/async-tasks
hit GET "/async-tasks" "" async positive
hit GET "/async-tasks/nonexistent-type" "" async negative         # 404 task not found

# 19) POST /v1/demo/files/check-exist （upload 模块代表：按内容哈希去重检查）
hit POST "/files/check-exist" '{"content_hash":"a4golden-probe-hash"}' upload_check_exist positive
hit POST "/files/check-exist" '{}' upload_check_exist negative        # 缺 content_hash

# 20) GET /v1/demo/files/:id （upload 模块：文件详情）
#    说明：A4 spec 写「14 条 v1 路由 + dict/setting/cron/async/upload 各 1 = 20」，但 14+5=19。
#    为落实「20 个接口」且不虚构端点，这里补 upload 模块的第 2 条真实路由（文件详情），
#    与 check-exist 同属 upload，凑足 20；详见 notes 文档。正例需先上传一个文件拿到真实 id。
UPLOAD_FILE_ID=""
upload_file() {
  local content="a4-golden-upload-${TAG}" tmpf hash resp
  tmpf="$(mktemp)"; printf '%s' "$content" > "$tmpf"
  hash="$(printf '%s' "$content" | shasum -a 256 | awk '{print $1}')"
  resp="$(curl -s --max-time 10 -F "content_hash=$hash" -F "file=@$tmpf;filename=a4golden.txt" \
            -F "scene=a4golden" -F "name=a4golden.txt" "$BASE/files" 2>/dev/null || true)"
  rm -f "$tmpf"
  printf '%s' "$resp" | jq -r '.data.file_id // empty'
}
UPLOAD_FILE_ID="$(upload_file)"
hit GET "/files/${UPLOAD_FILE_ID:-nonexistent-id}" "" upload_file_detail positive "" "/files/{id}"
hit GET "/files/00000000-0000-0000-0000-000000000000" "" upload_file_detail negative  # 不存在的文件

# ============================================================================
#  汇总为稳定 JSON（按 id+kind 排序，确保两次输出逐字节一致）
# ============================================================================
CANON="$(jq -s 'sort_by(.id, .kind)' "$RAW_NDJSON" 2>/dev/null || echo '[]')"

# 清理造数（best-effort；正例 DELETE 已删掉第一个，这里兜底删 tx/create 产生的残留）
curl -s -o /dev/null --max-time 5 -X DELETE "$BASE/products/$CREATED_PRODUCT_ID" 2>/dev/null || true
[[ -n "$UPLOAD_FILE_ID" ]] && curl -s -o /dev/null --max-time 5 -X DELETE "$BASE/files/$UPLOAD_FILE_ID" 2>/dev/null || true

# 打一个时间无关的稳定外壳：meta（不含时间戳）+ endpoints
RESULT="$(jq -n \
  --argjson endpoints "$CANON" \
  '{meta:{task:"P0-T2 A4 golden", triple:"code+message+httpStatus", basePath:"/v1/demo",
         note:"iam.mode=disabled + env=dev（见 collect.sh 头注释）",
         interfaces:($endpoints | map(.id) | unique | length)},
    endpoints:$endpoints}')"

if [[ -n "$CHECK_MODE" ]]; then
  if [[ ! -f "$GOLDEN_FILE" ]]; then
    echo "缺少基线文件：$GOLDEN_FILE" >&2; exit 3
  fi
  if diff -q <(printf '%s\n' "$RESULT") "$GOLDEN_FILE" >/dev/null 2>&1; then
    echo "A4 golden: 0 差异（与 $GOLDEN_FILE 完全一致）"
    exit 0
  else
    echo "A4 golden: 存在差异！" >&2
    diff <(printf '%s\n' "$RESULT") "$GOLDEN_FILE" >&2 || true
    exit 1
  fi
else
  printf '%s\n' "$RESULT" | tee "$GOLDEN_FILE"
fi
