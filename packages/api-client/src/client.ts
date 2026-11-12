/**
 * packages/api-client —— P5-T2 契约驱动生成的 TS 客户端（试点）。
 *
 * 结构：
 *   - `schema.ts` 由 openapi-typescript 从 `contracts/openapi/openapi.json` **自动生成**（勿手改）。
 *   - 本文件用 `openapi-fetch` 基于 `schema.ts` 的 `paths` 类型构造**强类型** HTTP 客户端。
 *
 * 试点范围：仅 1 个接口 `GET /v1/demo/products`（operationId `listProducts`）。后续扩展只需重跑
 * `pnpm --filter @ark-go-starter/api-client generate` 更新 `schema.ts`，再按需增补封装函数。
 *
 * 约束（Q6）：本包只消费**生成物**；不得 import `contracts/` 源文件（源 spec 仅被工具读取/校验）。
 */
import createClient from 'openapi-fetch'
import type { paths } from './schema'

export type { paths, components, operations } from './schema'

/** 默认后端基址（本地 dev，与 openapi.json 的 servers[0].url 一致）。 */
export const DEFAULT_BASE_URL = 'http://localhost:8099'

/**
 * 创建强类型 API 客户端。
 *
 * @param baseUrl 后端基址，缺省 {@link DEFAULT_BASE_URL}
 * @param fetch   可选自定义 fetch 实现（便于注入鉴权头、测试 mock 等）
 */
export function createApiClient(
  baseUrl: string = DEFAULT_BASE_URL,
  fetch?: typeof globalThis.fetch,
) {
  return createClient<paths>({ baseUrl, ...(fetch ? { fetch } : {}) })
}

/** 默认客户端实例（基址为本地 dev）。 */
export const apiClient = createApiClient()

/**
 * 商品分页列表（试点接口）：`GET /v1/demo/products`。
 *
 * @param query 查询参数（categoryID / name / page / pageSize，均可选）
 * @returns openapi-fetch 的 `{ data, error, response }`
 */
export function listProducts(query?: paths['/v1/demo/products']['get']['parameters']['query']) {
  return apiClient.GET('/v1/demo/products', { params: { query } })
}
