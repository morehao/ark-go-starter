import { test, expect } from '../fixtures/base'

/**
 * 异步任务页 · 全流程。
 *
 * 页面能力（读 pages/Async/index.tsx 确认）：分页列表、运行记录抽屉、启停、刷新。
 * 与定时任务一致的两个真实约束：
 * 1) 任务类型只能由后端代码 `Server.Register` 注册（ginasync 不提供 POST 建任务），e2e 只消费
 *    dev 环境注册的示例任务类型；
 * 2) 页面没有 mark-stale / 清理 runs 按钮（后端有 API、UI 未暴露），归后端 e2e 覆盖。
 * 注意异步任务按 `:type` 维度操作，页面用 record.id 作为 type 传给后端（PATCH /async-tasks/{type}/status）。
 *
 * 断言稳定性：antd 两字按钮插空格 → /禁\s*用/、/启\s*用/、/刷\s*新/；抽屉用 .ant-drawer 限定。
 */
async function gotoAsync(page: import('@playwright/test').Page) {
  await page.goto('/')
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/demo/async-tasks') && r.request().method() === 'GET',
    ),
    page.getByRole('menuitem', { name: '异步任务' }).click(),
  ])
  expect(resp.status(), '异步任务列表接口应为 200').toBe(200)
  const body = (await resp.json()) as { code: number; data: { list: unknown[]; total: number } }
  expect(body.code).toBe(0)
  await expect(page.getByRole('heading', { name: '异步任务', level: 5 })).toBeVisible()
  return body.data
}

test.describe('异步任务 · 全流程', () => {
  test('列表：列头与数据行渲染，且后端有已注册任务类型', async ({ page }) => {
    const data = await gotoAsync(page)
    expect(data.total, '后端应注册至少 1 个异步任务类型（dev 示例任务）').toBeGreaterThan(0)

    const table = page.locator('.ant-table').first()
    for (const h of ['名称', '描述', '状态', '操作']) {
      await expect(table.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    await expect(firstRow.locator('.ant-tag')).toBeVisible()
    await expect(firstRow.getByRole('button', { name: /运行\s*记录/ })).toBeVisible()
  })

  test('运行记录抽屉：打开 → 表头渲染 → 关闭', async ({ page }) => {
    await gotoAsync(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    const taskName = (await firstRow.locator('td').first().innerText()).trim()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /运行\s*记录/ }).click(),
    ])
    expect(((await runsResp.json()) as { code: number }).code).toBe(0)

    const drawer = page.locator('.ant-drawer')
    await expect(page.locator('.ant-drawer-open')).toHaveCount(1)
    await expect(drawer.getByText(`运行记录 · ${taskName}`, { exact: true })).toBeVisible()
    for (const h of ['队列', '状态', '重试', '开始时间', '耗时(ms)', '错误信息']) {
      await expect(drawer.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }

    await drawer.locator('.ant-drawer-close').click()
    await expect(page.locator('.ant-drawer-open')).toHaveCount(0)
  })

  test('启停：切换状态 → 提示成功 → Tag 翻转 → 再切回原状态', async ({ page }) => {
    await gotoAsync(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    const tag = firstRow.locator('.ant-tag')
    const before = (await tag.innerText()).trim()
    expect(['启用', '禁用'], `未知任务状态：${before}`).toContain(before)

    const firstBtn = before === '启用' ? /禁\s*用/ : /启\s*用/
    const [patchResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      firstRow.getByRole('button', { name: firstBtn }).click(),
    ])
    expect(((await patchResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-message')).toContainText('状态已更新')
    await expect(tag).not.toHaveText(before)
    const flipped = (await tag.innerText()).trim()

    // 恢复初始状态
    const secondBtn = flipped === '启用' ? /禁\s*用/ : /启\s*用/
    const [restoreResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      firstRow.getByRole('button', { name: secondBtn }).click(),
    ])
    expect(((await restoreResp.json()) as { code: number }).code).toBe(0)
    await expect(tag).toHaveText(before)
  })

  test('刷新：重新请求任务列表', async ({ page }) => {
    await gotoAsync(page)
    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/async-tasks') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    expect(((await refreshResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-table-tbody .ant-table-row').first()).toBeVisible()
  })
})
