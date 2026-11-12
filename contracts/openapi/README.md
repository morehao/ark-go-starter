# contracts/openapi — HTTP 契约单一事实源（P5 试点）

本目录是 HTTP 契约的**单一事实源**（OpenAPI）。约束（Q6 定案）：**任何模块不得 `import contracts`**；契约只以「生成物」进入 `libs`（TS 客户端 / Go client），契约源文件本身只被工具读取与校验。

## 当前内容（P5-T1 试点）

- `openapi.json` — OpenAPI **3.1.0** 试点 spec，仅含 1 个接口：`GET /v1/demo/products`（商品分页列表），用于验证 `swag → OpenAPI 3.1` 工具链可行。

## 试点工具链（可复现）

前置：Go（`swag` v1.16.4 已装）、node 26 / pnpm 12、npx 可用。Go 侧命令在仓库根用 `GOCACHE` 隔离。

```bash
# 1) swag 从源码注解生成 swagger.json（2.0），输出到临时目录
swag init --parseDependency --parseInternal --parseDependencyLevel 2 \
  -g app.go --dir apps/api --output .dsh/tmp/swag-out --outputTypes json --instanceName demo

# 2) （试点裁剪）仅保留 GET /v1/demo/products 及其 $ref 定义 → pilot-products-swagger.json（2.0）
#    （见 .dsh/docs/notes/2026-11-12-p5t1-openapi-pilot.md 的试点脚本）

# 3) swagger 2.0 → OpenAPI 3.0
npx swagger2openapi@7 .dsh/tmp/swag-out/pilot-products-swagger.json -o .dsh/tmp/swag-out/pilot-products-openapi30.json

# 4) OpenAPI 3.0 → 3.1（bump openapi: 3.1.0，nullable→type union；试点补 servers/security/operationId/4xx）

# 5) 校验（两种可靠校验器）
npx @apidevtools/swagger-cli@4 validate contracts/openapi/openapi.json   # → is valid
npx @redocly/cli@latest lint contracts/openapi/openapi.json              # → Woohoo! Your API description is valid.
```

> npx 在受限沙箱下需把 npm 缓存指向可写目录：`npm_config_cache=$PWD/.dsh/tmp/npm-cache npx --cache ...`。

## 后续（P5-T2 / P5-T3）

- P5-T2（R2）：`libs/ts/api-client` 由本目录生成物替换（保留 `legacy/` 一个周期）。
- P5-T3（R1）：`libs/go/client` 生成 + `apps/api` 接入。
- P5-T4（R3）：`make contracts-verify` 进 CI（生成物 diff 即阻断）。
