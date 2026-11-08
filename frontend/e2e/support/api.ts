/**
 * 直连后端的 REST 辅助，用于造数与清理。
 *
 * e2e 数据统一用 e2e- 前缀 + 时间戳，保证唯一；清理时按前缀或 id 删除，避免污染真实库。
 * baseURL 指向后端根（默认 http://localhost:8099），不经过前端代理，减少一层依赖。
 */
export const API_BASE = process.env.E2E_API_BASE ?? 'http://localhost:8099'

export interface ProductCreateReq {
  name: string
  categoryID: number
  price: number
}

export interface ProductItem {
  productID: number
  name: string
  categoryID: number
  price: number
}

interface ApiResp<T> {
  code: number
  msg: string
  data: T
}

/** 生成唯一测试名，前缀 e2e-。 */
export function uniqueName(prefix = 'e2e-product'): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e4)}`
}

async function call<T>(method: string, path: string, body?: unknown): Promise<ApiResp<T>> {
  const resp = await fetch(`${API_BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await resp.text()
  if (!resp.ok) {
    throw new Error(`${method} ${path} -> HTTP ${resp.status}: ${text}`)
  }
  return JSON.parse(text) as ApiResp<T>
}

/** 通过接口创建商品，返回商品 ID。 */
export async function createProduct(req: ProductCreateReq): Promise<number> {
  const res = await call<{ productID: number }>('POST', '/v1/demo/products', req)
  if (res.code !== 0) throw new Error(`create product failed: ${res.msg}`)
  return res.data.productID
}

/** 通过接口删除商品（软删）。 */
export async function deleteProduct(id: number): Promise<void> {
  await call<unknown>('DELETE', `/v1/demo/products/${id}`)
}

/** 列出商品（分页）。 */
export async function listProducts(page = 1, pageSize = 50): Promise<{ list: ProductItem[]; total: number }> {
  const res = await call<{ list: ProductItem[]; total: number }>(
    'GET',
    `/v1/demo/products?page=${page}&pageSize=${pageSize}`,
  )
  return res.data
}

/** 清理所有名称以 e2e- 开头的商品（软删），用于 globalTeardown。 */
export async function cleanupE2EProducts(): Promise<void> {
  const { list } = await listProducts(1, 100)
  const targets = list.filter((p) => p.name.startsWith('e2e-'))
  await Promise.all(targets.map((p) => deleteProduct(p.productID).catch(() => undefined)))
}
