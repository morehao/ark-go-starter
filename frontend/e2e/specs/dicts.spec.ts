import { test, expect } from '../fixtures/base'
import {
  cleanupE2EDicts,
  createDictItem,
  createDictType,
  deleteDictType,
  listDictTypes,
  uniqueName,
} from '../support/api'

/**
 * 数据字典页 · 全流程。
 *
 * UI 实际能力（读 pages/Dicts/index.tsx 确认）：分页列表、查看项抽屉、启停、删除。
 * 注意：该页**没有**新建/编辑表单（后端 /dict-types 有 POST/PUT，但 UI 未暴露），
 * 因此「新建/编辑」用接口造数覆盖数据链路，UI 侧覆盖列表/抽屉/启停/删除拦截。
 *
 * 断言稳定性要点：
 * - antd 会在两个汉字间插空格（「删 除」「启 用」），按钮一律用 /删\s*除/ 这类正则；
 * - 状态 Tag 用 .ant-tag 限定，避免和同名按钮文本冲突；
 * - Popconfirm / Drawer 用 .ant-popconfirm、.ant-drawer 限定作用域，避免 strict mode 命中多个；
 * - 关键写操作都用 waitForResponse 等待真实请求完成，不靠 sleep。
 *
 * 数据：全部 e2e- 前缀 + 时间戳；用例内 UI 删除，afterEach 按 id 级联兜底清理。
 */
const PREFIX = 'e2e-dict'

/** 进入数据字典页，并等待首屏列表请求返回。 */
async function gotoDicts(page: import('@playwright/test').Page) {
  await page.goto('/')
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/demo/dict-types') && r.request().method() === 'GET',
    ),
    page.getByRole('menuitem', { name: '数据字典' }).click(),
  ])
  expect(resp.status(), '字典类型列表接口应为 200').toBe(200)
  await expect(page.getByRole('heading', { name: '数据字典', level: 5 })).toBeVisible()
}

/** 取某 code 对应的表格行。 */
function rowOf(page: import('@playwright/test').Page, code: string) {
  return page.getByRole('row', { name: new RegExp(code) })
}

test.describe('数据字典 · 全流程', () => {
  test.describe.configure({ mode: 'serial' })

  const createdTypeIDs: string[] = []

  test.beforeEach(async () => {
    // 兜底：上一轮意外残留的 e2e- 数据先清掉，避免干扰分页/行定位。
    await cleanupE2EDicts().catch(() => undefined)
  })

  test.afterEach(async () => {
    for (const id of createdTypeIDs.splice(0)) {
      await deleteDictType(id, true).catch(() => undefined)
    }
  })

  test('列表：类型行渲染编码/名称/状态/说明', async ({ page }) => {
    const code = uniqueName(PREFIX)
    const name = `${code}-name`
    const type = await createDictType({ code, name, description: 'e2e 字典说明' })
    createdTypeIDs.push(type.id)

    await gotoDicts(page)
    const row = rowOf(page, code)
    await expect(row).toBeVisible()
    await expect(row).toContainText(name)
    await expect(row).toContainText('e2e 字典说明')
    // 新建类型默认 enabled → 状态列 Tag 显示「启用」
    await expect(row.locator('.ant-tag')).toHaveText('启用')
    // 操作列三个按钮齐全（正则兼容 antd 自动插入的空格）
    await expect(row.getByRole('button', { name: /查看\s*项/ })).toBeVisible()
    await expect(row.getByRole('button', { name: /禁\s*用/ })).toBeVisible()
    await expect(row.getByRole('button', { name: /删\s*除/ })).toBeVisible()
  })

  test('启停：禁用 → 状态 Tag 变禁用；再启用 → 恢复', async ({ page }) => {
    const code = uniqueName(PREFIX)
    const type = await createDictType({ code })
    createdTypeIDs.push(type.id)

    await gotoDicts(page)
    const row = rowOf(page, code)
    await expect(row.locator('.ant-tag')).toHaveText('启用')

    // 禁用
    const [disableResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes(`/demo/dict-types/${type.id}/status`) && r.request().method() === 'PATCH',
      ),
      row.getByRole('button', { name: /禁\s*用/ }).click(),
    ])
    expect((await disableResp.json()).code, '启停接口业务码应为 0').toBe(0)
    await expect(page.locator('.ant-message')).toContainText('状态已更新')
    await expect(row.locator('.ant-tag')).toHaveText('禁用')

    // 再启用（此时按钮文案变成「启 用」）
    const [enableResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes(`/demo/dict-types/${type.id}/status`) && r.request().method() === 'PATCH',
      ),
      row.getByRole('button', { name: /启\s*用/ }).click(),
    ])
    expect((await enableResp.json()).code).toBe(0)
    await expect(row.locator('.ant-tag')).toHaveText('启用')

    // 接口侧复核：状态确实落库为 enabled
    const { list } = await listDictTypes(1, 100)
    expect(list.find((t) => t.id === type.id)?.status).toBe('enabled')
  })

  test('查看项抽屉：展示项列表与层级，含停用项', async ({ page }) => {
    const code = uniqueName(PREFIX)
    const name = `${code}-name`
    const type = await createDictType({ code, name })
    createdTypeIDs.push(type.id)
    const root = await createDictItem({ type_code: code, value: 'root', label: '根节点' })
    const child = await createDictItem({
      type_code: code,
      value: 'child',
      label: '子节点',
      parent_id: root.id,
    })
    await createDictItem({
      type_code: code,
      value: 'off',
      label: '停用节点',
      status: 'disabled',
    })

    await gotoDicts(page)
    const row = rowOf(page, code)

    const [itemsResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes(`/demo/dict-codes/${code}/items`) && r.request().method() === 'GET',
      ),
      row.getByRole('button', { name: /查看\s*项/ }).click(),
    ])
    expect(itemsResp.status()).toBe(200)

    const drawer = page.locator('.ant-drawer')
    await expect(drawer).toBeVisible()
    await expect(drawer.getByText(`字典项 · ${name}`, { exact: true })).toBeVisible()
    // 抽屉内表格列头（列顺序即页面实现）
    for (const h of ['码值', '展示文案', '层级', '排序', '状态', '说明']) {
      await expect(drawer.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }
    // 根/子项都在，且层级列真实反映树深度
    const rootRow = drawer.getByRole('row', { name: /root/ })
    const childRow = drawer.getByRole('row', { name: /child/ })
    await expect(rootRow).toContainText('根节点')
    await expect(rootRow).toContainText('1')
    await expect(childRow).toContainText('子节点')
    await expect(childRow).toContainText('2')
    // 停用项也展示（include_disabled=true），Tag 为禁用
    const offRow = drawer.getByRole('row', { name: /off/ })
    await expect(offRow.locator('.ant-tag')).toHaveText('禁用')
    // 层级断言用接口值兜底，避免仅靠文本匹配
    expect(child.level).toBe(2)

    // 关闭抽屉：antd v6 关闭后仍保留 .ant-drawer 根节点（仅移除 .ant-drawer-open），
    // 所以断言「open 状态类消失」而不是元素不可见。
    await drawer.locator('.ant-drawer-close').click()
    await expect(page.locator('.ant-drawer-open')).toHaveCount(0)
  })

  test('删除拦截：类型下仍有字典项时删除被拒绝，行保留', async ({ page }) => {
    const code = uniqueName(PREFIX)
    const type = await createDictType({ code })
    createdTypeIDs.push(type.id)
    await createDictItem({ type_code: code, value: 'kept', label: '保留项' })

    await gotoDicts(page)
    const row = rowOf(page, code)
    await row.getByRole('button', { name: /删\s*除/ }).click()

    const popconfirm = page.locator('.ant-popconfirm')
    await expect(popconfirm).toBeVisible()
    const [delResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes(`/demo/dict-types/${type.id}`) && r.request().method() === 'DELETE',
      ),
      popconfirm.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    // 后端约定业务错误也走 HTTP 200 + 非 0 code（golib gincontext.Fail）
    expect(delResp.status()).toBe(200)
    const body = (await delResp.json()) as { code: number; msg: string }
    expect(body.code, '有子项时删除必须失败').not.toBe(0)
    expect(body.msg).toMatch(/still has items/i)

    // UI 提示错误，且行仍在（未被乐观删除）
    await expect(page.locator('.ant-message')).toContainText(/still has items/i)
    await expect(row).toBeVisible()
    // 接口侧复核：类型确实还在
    const { list } = await listDictTypes(1, 100)
    expect(list.some((t) => t.id === type.id)).toBeTruthy()
  })

  test('删除：空类型删除成功，行消失', async ({ page }) => {
    const code = uniqueName(PREFIX)
    const type = await createDictType({ code })
    createdTypeIDs.push(type.id)

    await gotoDicts(page)
    const row = rowOf(page, code)
    await row.getByRole('button', { name: /删\s*除/ }).click()

    const popconfirm = page.locator('.ant-popconfirm')
    const [delResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes(`/demo/dict-types/${type.id}`) && r.request().method() === 'DELETE',
      ),
      popconfirm.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    expect((await delResp.json()).code).toBe(0)
    await expect(page.locator('.ant-message')).toContainText('删除成功')
    await expect(row).toHaveCount(0)

    const { list } = await listDictTypes(1, 100)
    expect(list.some((t) => t.id === type.id)).toBeFalsy()
    // 已在 UI 删除，无需兜底
    createdTypeIDs.length = 0
  })

  test('刷新：重新请求字典类型列表', async ({ page }) => {
    const code = uniqueName(PREFIX)
    const type = await createDictType({ code })
    createdTypeIDs.push(type.id)

    await gotoDicts(page)
    await expect(rowOf(page, code)).toBeVisible()

    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/dict-types') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    expect(((await refreshResp.json()) as { code: number }).code).toBe(0)
    await expect(rowOf(page, code)).toBeVisible()
  })

  test('分页：翻到第 2 页与切换每页条数都会重新请求', async ({ page }) => {
    const base = uniqueName(`${PREFIX}-page`)
    for (let i = 0; i < 11; i++) {
      const t = await createDictType({ code: `${base}-${i}` })
      createdTypeIDs.push(t.id)
    }

    await gotoDicts(page)

    // 第 1 页 10 条，存在第 2 页
    const page2 = page.locator('.ant-pagination-item-2')
    await expect(page2).toBeVisible()

    const [page2Resp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return u.includes('/demo/dict-types') && r.request().method() === 'GET' && u.includes('page=2')
      }),
      page2.click(),
    ])
    const page2Body = (await page2Resp.json()) as { code: number; data: { list: unknown[] } }
    expect(page2Body.code).toBe(0)
    expect(page2Body.data.list.length).toBeGreaterThan(0)
    await expect(page.locator('.ant-pagination-item-2')).toHaveClass(/ant-pagination-item-active/)

    // 切换每页条数 10 → 20
    await page.locator('.ant-pagination-options .ant-select').click()
    const [sizeResp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return u.includes('/demo/dict-types') && r.request().method() === 'GET' && u.includes('page_size=20')
      }),
      page
        .locator('.ant-select-dropdown:visible')
        .getByText(/20\s*条\/页/)
        .click(),
    ])
    expect((await sizeResp.json()).code).toBe(0)
    // 行数不超过新的 pageSize
    const rowCount = await page.locator('.ant-table-tbody .ant-table-row').count()
    expect(rowCount).toBeGreaterThan(0)
    expect(rowCount).toBeLessThanOrEqual(20)
  })
})
