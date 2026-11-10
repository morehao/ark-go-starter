import { test, expect } from '../fixtures/base'
import {
  cleanupE2ESettings,
  createSettingAPI,
  listSettings,
  uniqueName,
} from '../support/api'

/**
 * 配置中心页 · 全流程。
 *
 * UI 实际能力（读 pages/Settings/index.tsx 确认）：分页列表、新建（Modal 表单）、编辑（值/描述可改，
 * 分组/键名/值类型 disabled）、删除（Popconfirm 二次确认）、刷新。
 * 该页**没有**分组/状态筛选控件、也没有加密项展示（后端 Setting 有 encryption_mode，UI 未渲染），
 * 这些在后端 e2e 覆盖，前端侧只在报告里记录为 UI 缺口。
 *
 * 断言稳定性要点：antd 两字按钮会插空格（「确 定」「删 除」）→ 统一用 /确\s*定|OK/i 与 /删\s*除/；
 * Modal/Popconfirm 用 .ant-modal/.ant-popconfirm 或 getByRole('dialog') 限定作用域，避免 strict mode。
 *
 * 数据：分组/键名统一 e2e- 前缀 + 时间戳，afterEach 走接口按前缀清理（UI 删除的也兜底一遍）。
 */
const PREFIX = 'e2e-setting'

/** 进入配置中心页，等待首屏列表请求返回。 */
async function gotoSettings(page: import('@playwright/test').Page) {
  await page.goto('/')
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/demo/settings') && r.request().method() === 'GET',
    ),
    page.getByRole('menuitem', { name: '配置中心' }).click(),
  ])
  expect(resp.status()).toBe(200)
  await expect(page.getByRole('heading', { name: '配置中心', level: 5 })).toBeVisible()
}

/** 按分组名定位表格行（分组在用例内唯一）。 */
function rowOf(page: import('@playwright/test').Page, group: string) {
  return page.getByRole('row', { name: new RegExp(group) })
}

test.describe('配置中心 · 全流程', () => {
  test.describe.configure({ mode: 'serial' })

  test.beforeEach(async () => {
    await cleanupE2ESettings(PREFIX).catch(() => undefined)
  })

  test.afterEach(async () => {
    await cleanupE2ESettings(PREFIX).catch(() => undefined)
  })

  test('新建：表单填写 → 创建成功 → 列表出现新行（含值类型选择）', async ({ page }) => {
    const group = uniqueName(PREFIX)
    const key = `${group}-key`
    await gotoSettings(page)

    await page.getByRole('button', { name: /新建配置/ }).click()
    const modal = page.getByRole('dialog')
    await expect(modal.getByText('新建配置项', { exact: true })).toBeVisible()

    await modal.getByPlaceholder('请输入分组').fill(group)
    await modal.getByPlaceholder('请输入键名').fill(key)
    // 值类型默认 string，这里显式切成 json，覆盖 Select 交互
    await modal.locator('.ant-select').click()
    await page.locator('.ant-select-dropdown:visible').getByText('json', { exact: true }).click()
    await modal.getByPlaceholder('请输入配置值').fill('{"enabled":true}')
    await modal.getByPlaceholder('请输入描述').fill('e2e 配置说明')

    const [createResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/settings') && r.request().method() === 'POST',
      ),
      modal.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    expect((await createResp.json()).code, '创建接口业务码应为 0').toBe(0)

    await expect(modal).toBeHidden()
    await expect(page.locator('.ant-message')).toContainText('创建成功')

    const row = rowOf(page, group)
    await expect(row).toBeVisible()
    await expect(row).toContainText(key)
    await expect(row).toContainText('json')
    await expect(row).toContainText('{"enabled":true}')
    await expect(row.locator('.ant-tag')).toHaveText('启用')
    // 注意：表格没有「描述」列，描述只能在接口侧断言（见下）

    // 接口侧复核落库（顺带拿到 id 供兜底清理）
    const { list } = await listSettings(1, 100)
    const saved = list.find((s) => s.group === group && s.key === key)
    expect(saved, '配置项应已落库').toBeTruthy()
    expect(saved?.value_type).toBe('json')
    expect(saved?.description, '描述应一并落库').toBe('e2e 配置说明')
    expect(saved?.encryption_mode, '明文配置 encryption_mode 应为 plain').toBe('plain')
  })

  test('表单校验：分组/键名/值均为空时提交被拦截', async ({ page }) => {
    await gotoSettings(page)
    await page.getByRole('button', { name: /新建配置/ }).click()
    const modal = page.getByRole('dialog')
    await expect(modal.getByText('新建配置项', { exact: true })).toBeVisible()

    await modal.getByRole('button', { name: /确\s*定|OK/i }).click()
    await expect(modal.getByText('请输入分组')).toBeVisible()
    await expect(modal.getByText('请输入键名')).toBeVisible()
    await expect(modal.getByText('请输入配置值')).toBeVisible()
    await expect(modal).toBeVisible()

    await modal.getByRole('button', { name: /取\s*消|Cancel/i }).click()
    await expect(modal).toBeHidden()
  })

  test('删除：Popconfirm 确认后行消失且接口复核已删', async ({ page }) => {
    const group = uniqueName(PREFIX)
    const key = `${group}-key`
    const created = await createSettingAPI({ group, key, value: 'to-delete' })

    await gotoSettings(page)
    const row = rowOf(page, group)
    await expect(row).toBeVisible()

    await row.getByRole('button', { name: /删\s*除/ }).click()
    const popconfirm = page.locator('.ant-popconfirm')
    await expect(popconfirm).toBeVisible()
    const [delResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/demo/settings/${created.id}`) && r.request().method() === 'DELETE',
      ),
      popconfirm.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    expect((await delResp.json()).code).toBe(0)
    await expect(page.locator('.ant-message')).toContainText('删除成功')
    await expect(row).toHaveCount(0)

    const { list } = await listSettings(1, 100)
    expect(list.some((s) => s.id === created.id)).toBeFalsy()
  })

  test('刷新：重新拉取列表且保持数据', async ({ page }) => {
    const group = uniqueName(PREFIX)
    const key = `${group}-key`
    await createSettingAPI({ group, key, value: 'refresh-me' })

    await gotoSettings(page)
    await expect(rowOf(page, group)).toBeVisible()

    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/settings') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    const body = (await refreshResp.json()) as { code: number }
    expect(body.code).toBe(0)
    await expect(rowOf(page, group)).toBeVisible()
  })

  test('分页：第 2 页与每页条数切换触发新请求', async ({ page }) => {
    const base = uniqueName(`${PREFIX}-page`)
    for (let i = 0; i < 11; i++) {
      await createSettingAPI({ group: `${base}-${i}`, key: `${base}-${i}-key`, value: `v${i}` })
    }

    await gotoSettings(page)
    const page2 = page.locator('.ant-pagination-item-2')
    await expect(page2).toBeVisible()

    const [page2Resp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return u.includes('/demo/settings') && r.request().method() === 'GET' && u.includes('page=2')
      }),
      page2.click(),
    ])
    const body = (await page2Resp.json()) as { code: number; data: { list: unknown[] } }
    expect(body.code).toBe(0)
    expect(body.data.list.length).toBeGreaterThan(0)

    await page.locator('.ant-pagination-options .ant-select').click()
    const [sizeResp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return u.includes('/demo/settings') && r.request().method() === 'GET' && u.includes('page_size=20')
      }),
      page.locator('.ant-select-dropdown:visible').getByText(/20\s*条\/页/).click(),
    ])
    expect((await sizeResp.json()).code).toBe(0)
  })

  test('编辑：分组/键名/值类型只读，改值与描述后行更新', async ({ page }) => {
    const group = uniqueName(PREFIX)
    const key = `${group}-key`
    const created = await createSettingAPI({
      group,
      key,
      value_type: 'string',
      value: 'v1',
      description: 'old',
    })

    await gotoSettings(page)
    const row = rowOf(page, group)
    await expect(row).toContainText('v1')

    await row.getByRole('button', { name: /编\s*辑/ }).click()
    const modal = page.getByRole('dialog')
    await expect(modal.getByText('编辑配置项', { exact: true })).toBeVisible()

    // 回填 + 只读：编辑弹窗必须用该行现有值回填，且分组/键名/值类型 disabled
    // （历史 P1：Modal destroyOnClose 导致 setFieldsValue 丢失 → 三个字段空且 disabled，编辑永远失败）
    await expect(modal.getByPlaceholder('请输入分组')).toHaveValue(group)
    await expect(modal.getByPlaceholder('请输入分组')).toBeDisabled()
    await expect(modal.getByPlaceholder('请输入键名')).toHaveValue(key)
    await expect(modal.getByPlaceholder('请输入键名')).toBeDisabled()
    await expect(modal.locator('.ant-select')).toContainText('string')
    await expect(modal.locator('.ant-select')).toHaveClass(/ant-select-disabled/)
    // 值/描述可改，且必须回填为行内现有值
    await expect(modal.getByPlaceholder('请输入配置值')).toHaveValue('v1')
    await expect(modal.getByPlaceholder('请输入描述')).toHaveValue('old')
    await modal.getByPlaceholder('请输入配置值').fill('v2-updated')
    await modal.getByPlaceholder('请输入描述').fill('new-desc')

    const [updateResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/demo/settings/${created.id}`) && r.request().method() === 'PUT',
      ),
      modal.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    expect((await updateResp.json()).code).toBe(0)
    await expect(page.locator('.ant-message')).toContainText('更新成功')

    await expect(row).toContainText('v2-updated')
    const { list } = await listSettings(1, 100)
    const edited = list.find((s) => s.id === created.id)
    expect(edited?.value).toBe('v2-updated')
    expect(edited?.description, '描述更新应落库（表格无描述列，接口侧断言）').toBe('new-desc')
  })
})
