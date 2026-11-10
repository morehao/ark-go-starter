import { test, expect } from '../fixtures/base'

/**
 * 定时任务页 · 全流程。
 *
 * 页面能力（读 pages/Cron/index.tsx 确认）：分页列表、运行记录抽屉、启停、刷新。
 * 注意两个真实约束：
 * 1) 任务定义只能由后端代码 `Scheduler.Register` 注册（gincron 不提供 POST 建任务），
 *    所以 e2e 不造任务，只消费后端 dev 环境注册的示例任务；
 * 2) 页面上**没有** mark-stale / 清理 runs 的按钮（后端有 API、UI 未暴露），这两项归后端 e2e 覆盖，
 *    前端 e2e 只覆盖列表/抽屉/启停/刷新。
 *
 * 断言稳定性：antd 两字按钮插空格 → 用 /禁\s*用/、/启\s*用/、/刷\s*新/；
 * 抽屉作用域 .ant-drawer（antd v6 关闭后根节点仍在 DOM，用 .ant-drawer-open 判定开关状态）。
 */
async function gotoCron(page: import('@playwright/test').Page) {
  await page.goto('/')
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/demo/cron-tasks') && r.request().method() === 'GET',
    ),
    page.getByRole('menuitem', { name: '定时任务' }).click(),
  ])
  expect(resp.status(), '定时任务列表接口应为 200').toBe(200)
  const body = (await resp.json()) as { code: number; data: { list: unknown[]; total: number } }
  expect(body.code).toBe(0)
  await expect(page.getByRole('heading', { name: '定时任务', level: 5 })).toBeVisible()
  return body.data
}

test.describe('定时任务 · 全流程', () => {
  test('列表：列头与数据行渲染，且后端有已注册任务', async ({ page }) => {
    const data = await gotoCron(page)
    // 后端 dev 环境注册了示例任务；列表必须非空，否则启停/抽屉都不可测
    expect(data.total, '后端应注册至少 1 个定时任务（dev 示例任务）').toBeGreaterThan(0)

    const table = page.locator('.ant-table').first()
    for (const h of ['名称', '任务类型', 'Cron 表达式', '状态', '下次运行时间', '操作']) {
      await expect(table.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    await expect(firstRow.locator('.ant-tag')).toBeVisible()
    await expect(firstRow.getByRole('button', { name: /运行\s*记录/ })).toBeVisible()
  })

  test('运行记录抽屉：打开 → 表头渲染 → 关闭', async ({ page }) => {
    await gotoCron(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    const taskName = (await firstRow.locator('td').first().innerText()).trim()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/cron-tasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /运行\s*记录/ }).click(),
    ])
    expect((await runsResp.json()) as { code: number }).toMatchObject({ code: 0 })

    const drawer = page.locator('.ant-drawer')
    await expect(page.locator('.ant-drawer-open')).toHaveCount(1)
    await expect(drawer.getByText(`运行记录 · ${taskName}`, { exact: true })).toBeVisible()
    for (const h of ['开始时间', '结束时间', '耗时(ms)', '状态', '错误信息']) {
      await expect(drawer.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }

    await drawer.locator('.ant-drawer-close').click()
    await expect(page.locator('.ant-drawer-open')).toHaveCount(0)
  })

  test('启停：切换状态 → 提示成功 → Tag 翻转 → 再切回原状态', async ({ page }) => {
    await gotoCron(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    const tag = firstRow.locator('.ant-tag')
    const before = (await tag.innerText()).trim()
    expect(['启用', '禁用'], `未知任务状态：${before}`).toContain(before)

    // 第一次切换
    const firstBtn = before === '启用' ? /禁\s*用/ : /启\s*用/
    const [patchResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/cron-tasks\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      firstRow.getByRole('button', { name: firstBtn }).click(),
    ])
    expect((await patchResp.json()) as { code: number }).toMatchObject({ code: 0 })
    await expect(page.locator('.ant-message')).toContainText('状态已更新')
    await expect(tag).not.toHaveText(before)
    const flipped = (await tag.innerText()).trim()

    // 切回原状态，避免改变后端示例任务的初始态
    const secondBtn = flipped === '启用' ? /禁\s*用/ : /启\s*用/
    const [restoreResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/cron-tasks\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      firstRow.getByRole('button', { name: secondBtn }).click(),
    ])
    expect((await restoreResp.json()) as { code: number }).toMatchObject({ code: 0 })
    await expect(tag).toHaveText(before)
  })

  test('刷新：重新请求任务列表', async ({ page }) => {
    await gotoCron(page)
    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/cron-tasks') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    expect(((await refreshResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-table-tbody .ant-table-row').first()).toBeVisible()
  })
})
