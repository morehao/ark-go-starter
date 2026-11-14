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
 * UI 实际能力（读 pages/Settings/index.tsx 确认）：分页列表、分组/状态筛选
 * （GET /settings?group=&status=&page=&page_size=，group 为精确匹配）、新建（Modal 表单，可开加密存储）、
 * 编辑（值/描述可改，分组/键名/值类型 disabled）、启停（状态列 StatusSwitch →
 * PATCH /settings/{id}/status）、删除（Popconfirm 二次确认）、刷新；
 * 列表新增「加密模式」列展示 encryption_mode（plain/encrypted）。
 * 错误文案由 demo-web 的 utils/errorMessage.ts 统一本地化（本页暂无稳定触发后端错误的用例）。
 *
 * 断言稳定性要点：antd 两字按钮会插空格（「确 定」「删 除」）→ 统一用 /确\s*定|OK/i 与 /删\s*除/；
 * 状态列是 StatusSwitch（Switch 的启用/禁用两段文案同时在 DOM，textContent="启用禁用"）→
 * 一律断言 .ant-switch 的 aria-checked（true=启用）；启停点开关后还需在 Modal.confirm 点确定；
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
  await expect(page.getByRole('heading', { name: '配置中心', level: 4 })).toBeVisible()
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
    // 状态列是开关（aria-checked=true 即启用），不再是 Tag
    await expect(row.locator('.ant-switch')).toHaveAttribute('aria-checked', 'true')
    // 新增「加密模式」列：默认明文落库 → plain
    await expect(row).toContainText('plain')
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

  test('分组筛选：输入分组 → 请求带 group 参数 → 只剩该分组行', async ({ page }) => {
    const groupA = uniqueName(`${PREFIX}-grp`)
    const groupB = uniqueName(`${PREFIX}-grp`)
    await createSettingAPI({ group: groupA, key: `${groupA}-key`, value: 'v-a' })
    await createSettingAPI({ group: groupB, key: `${groupB}-key`, value: 'v-b' })

    await gotoSettings(page)
    await expect(rowOf(page, groupA)).toBeVisible()
    await expect(rowOf(page, groupB)).toBeVisible()

    const search = page.getByPlaceholder('按分组筛选（精确匹配）')
    await search.fill(groupA)
    const [filterResp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return (
          u.includes('/demo/settings') &&
          r.request().method() === 'GET' &&
          u.includes(`group=${groupA}`)
        )
      }),
      search.press('Enter'),
    ])
    const body = (await filterResp.json()) as {
      code: number
      data: { list: Array<{ group: string }>; total: number }
    }
    expect(body.code).toBe(0)
    expect(body.data.total, '该分组只有 1 条 e2e 数据').toBe(1)
    expect(body.data.list.every((s) => s.group === groupA)).toBeTruthy()

    await expect(rowOf(page, groupA)).toBeVisible()
    await expect(rowOf(page, groupB)).toHaveCount(0)

    // 重置后两条都回来
    const [resetResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/settings') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /重\s*置/ }).click(),
    ])
    expect((await resetResp.json()) as { code: number }).toMatchObject({ code: 0 })
    await expect(rowOf(page, groupA)).toBeVisible()
    await expect(rowOf(page, groupB)).toBeVisible()
  })

  test('启停：状态列开关二次确认 → 停用 → 再启用 → 恢复', async ({ page }) => {
    const group = uniqueName(PREFIX)
    const key = `${group}-key`
    await createSettingAPI({ group, key, value: 'toggle-me', description: '启停用例' })

    await gotoSettings(page)
    const row = rowOf(page, group)
    const sw = row.locator('.ant-switch')
    await expect(sw).toHaveAttribute('aria-checked', 'true')

    // 停用：点开关 → Modal.confirm 二次确认 → 点「确认禁用」才发 PATCH
    await sw.click()
    const disableDialog = page.locator('.ant-modal-confirm:visible')
    await expect(disableDialog).toContainText('禁用该配置项？')
    const [disableResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/settings\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      disableDialog.getByRole('button', { name: /确\s*认/ }).click(),
    ])
    expect((await disableResp.json()).code, '启停接口业务码应为 0').toBe(0)
    // 成功提示由 StatusSwitch 统一给出（停用→「已停用」，启用→「已恢复」）
    await expect(page.locator('.ant-message')).toContainText('已停用')
    await expect(sw).toHaveAttribute('aria-checked', 'false')

    // 接口侧复核：状态确实落库为 disabled，且值与描述未被启停接口改动
    const { list } = await listSettings(1, 100)
    const saved = list.find((s) => s.group === group && s.key === key)
    expect(saved?.status, '启停应落库为 disabled').toBe('disabled')
    expect(saved?.value, '启停不应改动配置值').toBe('toggle-me')
    expect(saved?.description, '启停不应改动描述').toBe('启停用例')

    // 再启用
    await sw.click()
    const enableDialog = page.locator('.ant-modal-confirm:visible')
    await expect(enableDialog).toContainText('启用该配置项？')
    const [enableResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/demo\/settings\/[^/]+\/status/.test(r.url()) && r.request().method() === 'PATCH',
      ),
      enableDialog.getByRole('button', { name: /确\s*认/ }).click(),
    ])
    expect((await enableResp.json()).code).toBe(0)
    await expect(page.locator('.ant-message')).toContainText('已恢复')
    await expect(sw).toHaveAttribute('aria-checked', 'true')
  })

  test('状态筛选：选择禁用 → 请求带 status=disabled → 只展示禁用项', async ({ page }) => {
    const groupOn = uniqueName(`${PREFIX}-on`)
    const groupOff = uniqueName(`${PREFIX}-off`)
    await createSettingAPI({ group: groupOn, key: `${groupOn}-key`, value: 'v-on' })
    await createSettingAPI({
      group: groupOff,
      key: `${groupOff}-key`,
      value: 'v-off',
      status: 'disabled',
    })

    await gotoSettings(page)
    await expect(rowOf(page, groupOn)).toBeVisible()
    await expect(rowOf(page, groupOff).locator('.ant-switch')).toHaveAttribute(
      'aria-checked',
      'false',
    )

    // 筛选区是页面内容卡里的第一个 Space（含分组搜索框），据此限定状态 Select，避免命中 Modal 内的 Select
    const filterBar = page
      .locator('.ant-space')
      .filter({ has: page.getByPlaceholder('按分组筛选（精确匹配）') })
    await filterBar.locator('.ant-select').click()
    const [filterResp] = await Promise.all([
      page.waitForResponse((r) => {
        const u = r.url()
        return (
          u.includes('/demo/settings') &&
          r.request().method() === 'GET' &&
          u.includes('status=disabled')
        )
      }),
      page.locator('.ant-select-dropdown:visible').getByText('禁用', { exact: true }).click(),
    ])
    const body = (await filterResp.json()) as {
      code: number
      data: { list: Array<{ status: string }> }
    }
    expect(body.code).toBe(0)
    expect(body.data.list.every((s) => s.status === 'disabled')).toBeTruthy()

    await expect(rowOf(page, groupOff)).toBeVisible()
    await expect(rowOf(page, groupOn)).toHaveCount(0)
  })

  test('加密模式：新建时开启加密存储 → 行展示 加密（encrypted）', async ({ page }) => {
    const group = uniqueName(PREFIX)
    const key = `${group}-key`

    await gotoSettings(page)
    await page.getByRole('button', { name: /新建配置/ }).click()
    const modal = page.getByRole('dialog')
    await modal.getByPlaceholder('请输入分组').fill(group)
    await modal.getByPlaceholder('请输入键名').fill(key)
    await modal.getByPlaceholder('请输入配置值').fill('secret-value')
    // 开启「加密存储」开关（默认关闭）
    await modal.getByRole('switch').click()

    const [createResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/demo/settings') && r.request().method() === 'POST',
      ),
      modal.getByRole('button', { name: /确\s*定|OK/i }).click(),
    ])
    expect((await createResp.json()) as { code: number }).toMatchObject({ code: 0 })
    await expect(modal).toBeHidden()
    await expect(page.locator('.ant-message')).toContainText('创建成功')

    const row = rowOf(page, group)
    await expect(row).toBeVisible()
    await expect(row).toContainText('加密（encrypted）')

    const { list } = await listSettings(1, 100)
    const saved = list.find((s) => s.group === group && s.key === key)
    expect(saved?.encryption_mode, '加密开关应落库为 encrypted').toBe('encrypted')
  })
})
