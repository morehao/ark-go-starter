import { test, expect } from '../fixtures/base'
import { listAsyncTaskRuns } from '../support/api'

/**
 * 异步任务页 · 全流程。
 *
 * 页面能力（读 pages/Async/index.tsx 确认）：分页列表、运行记录抽屉、启停、刷新；
 * 运行记录抽屉内提供两个运维动作（按 :type 维度）：
 * - 标记陈旧：POST /async-tasks/:type/runs/mark-stale（body `{ cutoff_seconds }`）
 * - 清理运行记录：DELETE /async-tasks/:type/runs（body `{ before_unix }`，经 Popconfirm 二次确认）
 * 错误文案由 demo-web 的 utils/errorMessage.ts 统一本地化。
 *
 * 与定时任务一致的两个真实约束：
 * 1) 任务类型只能由后端代码 `Server.Register` 注册（ginasync 不提供 POST 建任务），e2e 只消费
 *    dev 环境注册的示例任务类型；
 * 2) 运行记录没有写接口，mark-stale/cleanup 断言真实请求体契约 + affected 返回，
 *    并用「清理 10 年前」的边界交叉复核不会减少现有数据。
 * 注意异步任务按 `:type` 维度操作，页面用 record.id 作为 type 传给后端。
 *
 * 断言稳定性：antd 两字按钮插空格 → /刷\s*新/；状态列是 StatusSwitch，按 .ant-switch 的 aria-checked 断言；
 * 抽屉用 .ant-drawer-open 限定。
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
    // 状态列是 StatusSwitch（开关），不再是 Tag
    await expect(firstRow.locator('.ant-switch')).toBeVisible()
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

  test('启停：状态列开关二次确认 → 提示成功 → 开关翻转 → 再切回原状态', async ({ page }) => {
    await gotoAsync(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    // 状态列是 StatusSwitch（DESIGN §7.2），不是 Tag。
    // 注意：Switch 的启用/禁用两段文案同时存在于 DOM（textContent="启用禁用"），
    // 所以状态一律按 aria-checked 断言（true=启用），不能用可见文案。
    const sw = firstRow.locator('.ant-switch')
    const before = (await sw.getAttribute('aria-checked')) === 'true'

    // 点开关 → Modal.confirm 二次确认 → 点确定才发 PATCH
    await sw.click()
    const dialog = page.locator('.ant-modal-confirm:visible')
    await expect(dialog).toBeVisible()
    const [patchResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      dialog.getByRole('button', { name: /确\s*认/ }).click(),
    ])
    expect(((await patchResp.json()) as { code: number }).code).toBe(0)
    // 成功提示由 StatusSwitch 统一给出（停用→「已停用」，启用→「已恢复」）
    await expect(page.locator('.ant-message')).toContainText(/已停用|已恢复/)
    await expect(sw).toHaveAttribute('aria-checked', String(!before))

    // 恢复初始状态
    await sw.click()
    const [restoreResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      page.locator('.ant-modal-confirm:visible').getByRole('button', { name: /确\s*认/ }).click(),
    ])
    expect(((await restoreResp.json()) as { code: number }).code).toBe(0)
    await expect(sw).toHaveAttribute('aria-checked', String(before))
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

  test('标记陈旧：按分钟阈值 → POST mark-stale 契约正确并提示影响条数', async ({ page }) => {
    await gotoAsync(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /运行\s*记录/ }).click(),
    ])
    expect(((await runsResp.json()) as { code: number }).code).toBe(0)

    const drawer = page.locator('.ant-drawer-open')
    await expect(drawer).toBeVisible()
    // 抽屉工具栏第一个数字输入框 = 陈旧判定（分钟）
    await drawer.getByRole('spinbutton').first().fill('1440')

    const [markResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().endsWith('/runs/mark-stale') && r.request().method() === 'POST',
      ),
      drawer.getByRole('button', { name: /标记陈旧/ }).click(),
    ])
    const body = (await markResp.json()) as { code: number; data: { affected: number } }
    expect(body.code).toBe(0)
    expect(typeof body.data.affected).toBe('number')
    const reqBody = JSON.parse(markResp.request().postData() ?? '{}') as {
      cutoff_seconds?: number
    }
    expect(reqBody.cutoff_seconds, '请求体必须使用 cutoff_seconds（秒）').toBe(1440 * 60)

    await expect(page.locator('.ant-message')).toContainText(/已标记\s*\d+\s*条陈旧运行记录/)
  })

  test('清理运行记录：Popconfirm 确认 → DELETE 带 before_unix，且不误删近期数据', async ({ page }) => {
    const listBody = await gotoAsync(page)
    expect(listBody.total, '后端应注册至少 1 个异步任务类型').toBeGreaterThan(0)
    const type = (listBody.list[0] as { id: string }).id
    const before = await listAsyncTaskRuns(type)

    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/async-tasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /运行\s*记录/ }).click(),
    ])
    expect(((await runsResp.json()) as { code: number }).code).toBe(0)

    const drawer = page.locator('.ant-drawer-open')
    await expect(drawer).toBeVisible()
    // 第二个数字输入框 = 清理早于（天前）；填 3650 天保证不会删到任何近期数据
    await drawer.getByRole('spinbutton').nth(1).fill('3650')
    await drawer.getByRole('button', { name: /清理运行记录/ }).click()

    const popconfirm = page.locator('.ant-popconfirm')
    await expect(popconfirm).toBeVisible()
    const [delResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().endsWith('/runs') && r.request().method() === 'DELETE',
      ),
      popconfirm.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    const body = (await delResp.json()) as { code: number; data: { affected: number } }
    expect(body.code).toBe(0)
    expect(typeof body.data.affected).toBe('number')
    const reqBody = JSON.parse(delResp.request().postData() ?? '{}') as { before_unix?: number }
    expect(reqBody.before_unix, '请求体必须使用 before_unix（unix 秒）').toBeGreaterThan(0)

    await expect(page.locator('.ant-message')).toContainText(/已清理\s*\d+\s*条运行记录/)

    const after = await listAsyncTaskRuns(type)
    expect(after.total, '清理 3650 天前的记录不应删除现有数据').toBe(before.total)
  })
})
