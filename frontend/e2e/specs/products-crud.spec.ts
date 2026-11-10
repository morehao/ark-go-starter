import { test, expect } from '../fixtures/base'
import { uniqueName, createProduct, deleteProduct, listProducts } from '../support/api'

/**
 * 商品 CRUD 主链路：新建 → 列表出现 → 编辑 → 行更新 → 删除 → 行消失。
 *
 * 说明：
 * - 导航：当前应用用 state 切页，从默认的「商品管理」页开始即可。
 * - 表格行动态，用唯一名称定位行，再在该行内点「编辑/删除」，避免依赖自增 ID。
 * - 造数后登记到 createdIDs，afterEach 兜底清理，防止中断留下脏数据。
 */
test.describe('商品管理 · CRUD', () => {
  test.describe.configure({ mode: 'serial' })

  const createdIDs: number[] = []

  test.afterEach(async () => {
    // 兜底清理：删除本用例通过接口造出的商品（页面创建的由用例内 UI 删除）
    for (const id of createdIDs.splice(0)) {
      await deleteProduct(id).catch(() => undefined)
    }
  })

  test('新建 → 编辑 → 删除 全流程', async ({ page }) => {
    const name = uniqueName('e2e-product')
    const updatedName = `${name}-upd`

    await page.goto('/')
    // 等商品表格首屏加载完成（表头「商品管理」标题可见）
    await expect(page.getByRole('heading', { name: '商品管理', level: 5 })).toBeVisible()

    // 新建
    await page.getByTestId('product-create-btn').click()
    const modal = page.getByRole('dialog')
    await expect(modal.getByText('新建商品')).toBeVisible()
    await modal.getByPlaceholder('请输入商品名称').fill(name)
    await modal.getByPlaceholder('请输入分类 ID').fill('22')
    await modal.getByPlaceholder('请输入价格').fill('19.9')
    await modal.getByRole('button', { name: /确\s*定|OK/i }).click()

    // 成功后弹窗关闭，列表出现该行
    await expect(modal).toBeHidden()
    const row = page.getByRole('row', { name: new RegExp(name) })
    await expect(row).toBeVisible()
    await expect(row).toContainText('22')

    // 记录 ID 便于兜底清理
    const data = await listProducts(1, 100)
    const created = data.list.find((p) => p.name === name)
    if (created) createdIDs.push(created.productID)

    // 编辑（在行内点编辑）
    await row.getByRole('button', { name: '编辑' }).click()
    const editModal = page.getByRole('dialog')
    await expect(editModal.getByText('编辑商品')).toBeVisible()
    await editModal.getByPlaceholder('请输入商品名称').fill(updatedName)
    await editModal.getByPlaceholder('请输入价格').fill('29.9')
    await editModal.getByRole('button', { name: /确\s*定|OK/i }).click()
    await expect(editModal).toBeHidden()

    // 行名更新
    const updatedRow = page.getByRole('row', { name: new RegExp(updatedName) })
    await expect(updatedRow).toBeVisible()
    await expect(updatedRow).toContainText('29.9')

    // 删除（Popconfirm 二次确认）
    await updatedRow.getByRole('button', { name: '删除' }).click()
    await page.getByRole('button', { name: /确\s*定|OK/i }).click()

    // 行消失
    await expect(page.getByRole('row', { name: new RegExp(updatedName) })).toHaveCount(0)

    // 已删除则无需兜底清理
    createdIDs.length = 0
  })
})

/**
 * 商品管理 · 列表交互：分页 / 每页条数 / 刷新 / 删除取消 / 编辑表单回填。
 *
 * 造数：beforeAll 通过接口创建 11 条 e2e- 商品（保证第 2 页存在），afterAll 统一软删。
 * 断言都等真实请求（waitForResponse），不依赖行文本顺序，避免与并行 worker 的数据互相影响。
 */
test.describe('商品管理 · 列表交互', () => {
  test.describe.configure({ mode: 'serial' })

  const seededIDs: number[] = []

  test.beforeAll(async () => {
    const stamp = Date.now()
    for (let i = 0; i < 11; i++) {
      seededIDs.push(
        await createProduct({ name: `e2e-product-page-${stamp}-${i}`, categoryID: 7, price: i + 1 }),
      )
    }
  })

  test.afterAll(async () => {
    for (const id of seededIDs.splice(0)) {
      await deleteProduct(id).catch(() => undefined)
    }
  })

  test('分页：翻到第 2 页、切换每页条数都会重新请求', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: '商品管理', level: 5 })).toBeVisible()

    const page2 = page.locator('.ant-pagination-item-2')
    await expect(page2).toBeVisible()

    const [page2Resp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return u.includes('/v1/demo/products') && r.request().method() === 'GET' && u.includes('page=2')
      }),
      page2.click(),
    ])
    const body = (await page2Resp.json()) as { code: number; data: { list: unknown[] } }
    expect(body.code).toBe(0)
    expect(body.data.list.length).toBeGreaterThan(0)
    await expect(page.locator('.ant-pagination-item-2')).toHaveClass(/ant-pagination-item-active/)

    // 每页条数 10 → 20
    await page.locator('.ant-pagination-options .ant-select').click()
    const [sizeResp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return u.includes('/v1/demo/products') && r.request().method() === 'GET' && u.includes('page_size=20')
      }),
      page.locator('.ant-select-dropdown:visible').getByText(/20\s*条\/页/).click(),
    ])
    expect((await sizeResp.json()).code).toBe(0)
    const rowCount = await page.locator('.ant-table-tbody .ant-table-row').count()
    expect(rowCount).toBeGreaterThan(0)
    expect(rowCount).toBeLessThanOrEqual(20)
  })

  test('刷新：重新拉取列表并保持行可见', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: '商品管理', level: 5 })).toBeVisible()

    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/v1/demo/products') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    expect(((await refreshResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-table-tbody .ant-table-row').first()).toBeVisible()
  })

  test('删除取消：Popconfirm 点取消后行保留、接口未删', async ({ page }) => {
    // 只在「首屏第 1 页能渲染出来」的造数里挑目标，避免排序把目标挤到第 2 页导致定位不到
    const candidates = (await listProducts(1, 10)).list.filter((p) =>
      p.name.startsWith('e2e-product-page-'),
    )
    expect(candidates.length, '首屏应至少有一条 beforeAll 造的商品').toBeGreaterThan(0)
    const target = candidates[0]

    await page.goto('/')
    const row = page.getByRole('row', { name: new RegExp(`${target.name}(?![0-9])`) })
    await expect(row).toBeVisible()

    await row.getByRole('button', { name: /删\s*除/ }).click()
    const popconfirm = page.locator('.ant-popconfirm')
    await expect(popconfirm).toBeVisible()
    await popconfirm.getByRole('button', { name: /取\s*消|Cancel/i }).click()
    await expect(popconfirm).toBeHidden()

    await expect(row).toBeVisible()
    const after = await listProducts(1, 100)
    expect(after.list.some((p) => p.productID === target.productID), '取消后不应删除').toBeTruthy()
  })

  test('编辑：弹窗应回填该行现有名称/分类/价格', async ({ page }) => {
    const candidates = (await listProducts(1, 10)).list.filter((p) =>
      p.name.startsWith('e2e-product-page-'),
    )
    expect(candidates.length, '首屏应至少有一条 beforeAll 造的商品').toBeGreaterThan(0)
    const target = candidates[candidates.length - 1]

    await page.goto('/')
    const row = page.getByRole('row', { name: new RegExp(`${target.name}(?![0-9])`) })
    await expect(row).toBeVisible()
    await row.getByRole('button', { name: /编\s*辑/ }).click()

    const modal = page.getByRole('dialog')
    await expect(modal.getByText('编辑商品', { exact: true })).toBeVisible()
    // 编辑必须回填当前值，否则用户一保存就会把未改字段清空（历史 bug：Modal destroyOnClose 导致赋值丢失）
    await expect(modal.getByPlaceholder('请输入商品名称')).toHaveValue(target.name)
    await expect(modal.getByPlaceholder('请输入分类 ID')).toHaveValue(String(target.categoryID))
    await expect(modal.getByPlaceholder('请输入价格')).toHaveValue(String(target.price))

    await modal.getByRole('button', { name: /取\s*消|Cancel/i }).click()
    await expect(modal).toBeHidden()
  })
})
