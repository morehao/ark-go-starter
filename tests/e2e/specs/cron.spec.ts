import { test, expect } from '../fixtures/base'
import { listCronTaskRuns } from '../support/api'
import { descriptionsValue } from '../support/dom'

/**
 * 定时任务页 · 全流程。
 *
 * 页面能力（读 pages/Cron/index.tsx 确认）：分页列表、任务详情抽屉（GET /cron-tasks/:id）、
 * 运行记录抽屉、启停、刷新；运行记录抽屉内提供两个运维动作：
 * - 标记陈旧：POST /cron-tasks/:id/runs/mark-stale（body `{ cutoff_seconds }`）
 * - 清理运行记录：DELETE /cron-tasks/:id/runs（body `{ before_unix }`，经 Popconfirm 二次确认）
 * 错误文案由 demo-web 的 utils/errorMessage.ts 统一本地化。
 *
 * 两个真实约束：
 * 1) 任务定义只能由后端代码 `Scheduler.Register` 注册（gincron 不提供 POST 建任务），
 *    所以 e2e 不造任务，只消费后端 dev 环境注册的示例任务；
 * 2) 运行记录也没有写接口，因此 mark-stale/cleanup 用「接口侧交叉复核」：
 *    断言真实请求体契约（cutoff_seconds / before_unix）与 affected 返回，且清理 10 年前的记录不减少数据。
 *
 * 断言稳定性：antd 两字按钮插空格 → 用 /禁\s*用/、/启\s*用/、/刷\s*新/、/详\s*情/；
 * 抽屉作用域用 .ant-drawer-open（antd v6 关闭后根节点仍在 DOM，用它判定开关状态）。
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

  test('任务详情抽屉：GET 任务详情 → Descriptions 渲染任务字段', async ({ page }) => {
    await gotoCron(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()
    const taskName = (await firstRow.locator('td').first().innerText()).trim()

    const [detailResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          /\/demo\/cron-tasks\/[^/?]+$/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /详\s*情/ }).click(),
    ])
    const body = (await detailResp.json()) as {
      code: number
      data: { id: string; name: string; spec: string; task_type: string }
    }
    expect(body.code).toBe(0)

    const drawer = page.locator('.ant-drawer-open')
    await expect(drawer).toBeVisible()
    await expect(drawer.getByText(`任务详情 · ${taskName}`, { exact: true })).toBeVisible()
    for (const label of ['任务 ID', '名称', '任务类型', 'Cron 表达式', '状态', '说明']) {
      await expect(drawer.getByText(label, { exact: true })).toBeVisible()
    }
    // 详情内容与接口返回一致（不是列表行里的旧值）
    await expect(descriptionsValue(drawer, '任务 ID')).toHaveText(body.data.id)
    await expect(descriptionsValue(drawer, '名称')).toHaveText(taskName)
    await expect(descriptionsValue(drawer, 'Cron 表达式')).toHaveText(body.data.spec)
    await expect(descriptionsValue(drawer, '任务类型')).toHaveText(body.data.task_type)
    await expect(drawer.locator('.ant-tag')).toHaveText(/启用|禁用/)

    await drawer.locator('.ant-drawer-close').click()
    await expect(page.locator('.ant-drawer-open')).toHaveCount(0)
  })

  test('标记陈旧：按分钟阈值 → POST mark-stale 契约正确并提示影响条数', async ({ page }) => {
    await gotoCron(page)
    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/cron-tasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /运行\s*记录/ }).click(),
    ])
    expect((await runsResp.json()) as { code: number }).toMatchObject({ code: 0 })

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
    // 契约：cutoff_seconds 必填且 = 分钟 * 60（gincron markStaleRequest）
    const reqBody = JSON.parse(markResp.request().postData() ?? '{}') as {
      cutoff_seconds?: number
    }
    expect(reqBody.cutoff_seconds, '请求体必须使用 cutoff_seconds（秒）').toBe(1440 * 60)

    await expect(page.locator('.ant-message')).toContainText(/已标记\s*\d+\s*条陈旧运行记录/)
  })

  test('清理运行记录：Popconfirm 确认 → DELETE 带 before_unix，且不误删近期数据', async ({ page }) => {
    const listBody = await gotoCron(page)
    expect(listBody.total, '后端应注册至少 1 个定时任务').toBeGreaterThan(0)
    const taskID = (listBody.list[0] as { id: string }).id
    const before = await listCronTaskRuns(taskID)

    const firstRow = page.locator('.ant-table-tbody .ant-table-row').first()
    await expect(firstRow).toBeVisible()

    const [runsResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/cron-tasks\/[^/]+\/runs/.test(r.url()) && r.request().method() === 'GET',
      ),
      firstRow.getByRole('button', { name: /运行\s*记录/ }).click(),
    ])
    expect((await runsResp.json()) as { code: number }).toMatchObject({ code: 0 })

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
    // 契约：DELETE 用 JSON body { before_unix }（不是 query 参数）
    const reqBody = JSON.parse(delResp.request().postData() ?? '{}') as { before_unix?: number }
    expect(reqBody.before_unix, '请求体必须使用 before_unix（unix 秒）').toBeGreaterThan(0)

    await expect(page.locator('.ant-message')).toContainText(/已清理\s*\d+\s*条运行记录/)

    // 交叉复核：清理 10 年前的边界不会减少现有运行记录
    const after = await listCronTaskRuns(taskID)
    expect(after.total, '清理 3650 天前的记录不应删除现有数据').toBe(before.total)
  })
})
