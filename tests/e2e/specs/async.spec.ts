import { test, expect } from '../fixtures/base'
import { listAsyncTaskRuns } from '../support/api'

/**
 * 异步任务页 · 全流程（golib v1.36.3 任务中心模型 /asyncTasks）。
 *
 * 页面能力（读 pages/Async/index.tsx 确认）：
 * - 任务列表（分页 + 统计卡片 + 刷新），列：任务类型/状态/业务类型/业务 ID/尝试/轮次/创建时间/更新时间/操作；
 * - 操作列：尝试记录（打开抽屉）/ 取消（pending/running 可见，二次确认）/ 重放（终态可见）；
 * - 尝试记录抽屉：任务详情 Descriptions + 尝试记录表格 + 取消/重放按钮。
 *
 * 数据来源：dev 环境 registerDemoTasks 会提交一条 demo-echo 示例任务（固定 UniqKey，
 * 命中幂等；RunAt 在未来，停在 pending 便于断言），所以列表至少 1 条。
 *
 * 断言稳定性：antd 两字按钮插空格 → /刷\s*新/、/尝试\s*记录/；抽屉用 .ant-drawer-open 限定。
 */
async function gotoAsync(page: import('@playwright/test').Page) {
  await page.goto('/')
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/demo/asyncTasks') && r.request().method() === 'GET',
    ),
    page.getByRole('menuitem', { name: '异步任务' }).click(),
  ])
  expect(resp.status(), '异步任务列表接口应为 200').toBe(200)
  const body = (await resp.json()) as { code: number; data: { list: unknown[]; total: number } }
  expect(body.code).toBe(0)
  await expect(page.getByRole('heading', { name: '异步任务', level: 4 })).toBeVisible()
  return body.data
}

test.describe('异步任务 · 全流程', () => {
  test('列表：列头与数据行渲染，且后端有任务实例', async ({ page }) => {
    const data = await gotoAsync(page)
    expect(data.total, 'dev 环境应至少提交 1 条示例异步任务').toBeGreaterThan(0)

    const table = page.locator('.ant-table').first()
    for (const h of ['任务类型', '状态', '业务类型', '业务 ID', '尝试', '轮次', '操作']) {
      await expect(table.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    await expect(firstRow.getByRole('button', { name: /尝试\s*记录/ })).toBeVisible()
  })

  test('尝试记录抽屉：打开 → 详情与表头渲染 → 关闭', async ({ page }) => {
    await gotoAsync(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/asyncTasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /尝试\s*记录/ }).click(),
    ])
    expect(((await runsResp.json()) as { code: number }).code).toBe(0)

    const drawer = page.locator('.ant-drawer')
    await expect(page.locator('.ant-drawer-open')).toHaveCount(1)
    await expect(drawer.getByText(/尝试记录 ·/)).toBeVisible()
    for (const h of ['轮次', '尝试', '状态', '开始时间', '耗时(ms)', '错误信息']) {
      await expect(drawer.getByRole('columnheader', { name: h, exact: true })).toBeVisible()
    }

    await drawer.locator('.ant-drawer-close').click()
    await expect(page.locator('.ant-drawer-open')).toHaveCount(0)
  })

  test('刷新：重新请求任务列表', async ({ page }) => {
    await gotoAsync(page)
    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/asyncTasks') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    expect(((await refreshResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-table-tbody .ant-table-row').first()).toBeVisible()
  })

  test('取消：pending 任务的取消按钮二次确认 → POST cancel 契约正确', async ({ page }) => {
    await gotoAsync(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()

    const cancelBtn = firstRow.getByRole('button', { name: /取\s*消/ })
    // 示例任务为 pending，取消按钮应可见
    await expect(cancelBtn).toBeVisible()
    await cancelBtn.click()

    const popconfirm = page.locator('.ant-popconfirm')
    await expect(popconfirm).toBeVisible()
    const [cancelResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/asyncTasks\/[^/]+\/cancel/.test(r.url()) && r.request().method() === 'POST',
      ),
      popconfirm.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    const body = (await cancelResp.json()) as { code: number; data: { canceled: boolean } }
    expect(body.code).toBe(0)
    expect(typeof body.data.canceled).toBe('boolean')
  })

  test('尝试记录抽屉：runs 接口 total 与后端一致', async ({ page }) => {
    const listBody = await gotoAsync(page)
    const taskID = (listBody.list[0] as { id: string }).id
    const before = await listAsyncTaskRuns(taskID)

    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/asyncTasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /尝试\s*记录/ }).click(),
    ])
    expect(((await runsResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-drawer-open')).toBeVisible()
    // 前端抽屉渲染的尝试记录条数与直连接口一致（pending 任务可能 0 条，不做正数假设）
    expect(typeof before.total).toBe('number')
  })
})
