import { test, expect } from '../fixtures/base'
import {
  API_BASE,
  createProduct,
  createProductInTx,
  deleteProduct,
  getProduct,
  listProducts,
  uniqueName,
} from '../support/api'

/**
 * 商品管理 · 详情与事务接口 + 全页面 hash 路由覆盖。
 *
 * 本 spec 补齐既有用例未覆盖的两个后端契约与一个前端导航场景：
 *   1. GET  /v1/demo/products/{id}      —— 商品详情（前端无详情弹窗，用直连接口校验契约与数据一致性）
 *   2. POST /v1/demo/products/tx        —— 事务创建（带请求头治理字段，验证事务链路可用）
 *   3. 前端 hash 路由：非法 hash 回落商品管理页；合法 hash 直达对应页
 */
test.describe('商品管理 · 详情与事务接口', () => {
  const createdIDs: number[] = []

  test.afterEach(async () => {
    await Promise.all(createdIDs.splice(0).map((id) => deleteProduct(id).catch(() => undefined)))
  })

  test('详情接口：GET /products/{id} 返回与创建一致的全量字段', async () => {
    const name = uniqueName('e2e-product-detail')
    const id = await createProduct({ name, categoryID: 7, price: 12.5 })
    createdIDs.push(id)

    const detail = await getProduct(id)
    expect(Number(detail.productID)).toBe(id)
    expect(detail.name).toBe(name)
    expect(Number(detail.categoryID)).toBe(7)
    expect(Number(detail.price)).toBe(12.5)
  })

  test('详情接口：不存在的 ID 返回业务错误码（非 200 成功）', async () => {
    const resp = await fetch(`${API_BASE}/v1/demo/products/999999999`)
    const body = (await resp.json()) as { code: number }
    // 兼容实现：不存在时可能 200+code!=0 或 4xx；两者都不能是「成功返回数据」。
    expect(resp.status === 200 ? body.code !== 0 : resp.status >= 400).toBeTruthy()
  })

  test('事务创建：POST /products/tx 一次建两条并在列表中可查', async () => {
    const first = uniqueName('e2e-product-tx-a')
    const second = uniqueName('e2e-product-tx-b')
    const ids = await createProductInTx(first, second)
    expect(ids.length, '事务创建应返回两条商品ID').toBe(2)
    createdIDs.push(...ids)

    const { list } = await listProducts(1, 100)
    expect(list.some((p) => p.productID === ids[0]), '第一条事务记录应出现在列表中').toBeTruthy()
    expect(list.some((p) => p.productID === ids[1]), '第二条事务记录应出现在列表中').toBeTruthy()
  })
})

test.describe('前端 hash 路由', () => {
  // 说明：前端仅在初始化时读取 location.hash（未监听 hashchange），
  // 因此每个用例都通过「首次完整加载带 hash 的 URL」来验证直达，不能在同一页面内二次改 hash。

  test('非法 hash 回落到商品管理页', async ({ page }) => {
    await page.goto('/#not-a-real-page')
    await expect(page.getByRole('heading', { name: '商品管理', level: 4 })).toBeVisible()
  })

  test('合法 hash #dicts 直达数据字典页', async ({ page }) => {
    await page.goto('/#dicts')
    await expect(page.getByRole('heading', { name: '数据字典', level: 4 })).toBeVisible()
  })

  test('合法 hash #health 直达健康检查页', async ({ page }) => {
    await page.goto('/#health')
    await expect(page.getByRole('heading', { name: '健康检查', level: 4 })).toBeVisible()
  })
})
