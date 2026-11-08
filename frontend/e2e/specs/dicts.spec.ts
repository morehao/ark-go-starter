import { test, expect } from '../fixtures/base'

/**
 * 数据字典页冒烟：切换到该页后标题、说明、表头渲染正常。
 * 数据来自 /v1/dict-types（在部分运行方式下可能返回 404），因此只断言页面骨架，
 * 不依赖具体数据行，避免把后端模块缺失误判为前端缺陷。
 */
test.describe('数据字典 · 页面骨架', () => {
  test('标题与表头渲染', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('menuitem', { name: '数据字典' }).click()
    await expect(page.getByRole('heading', { name: '数据字典', level: 5 })).toBeVisible()

    // 说明文案可见
    await expect(page.getByText('字典类型分页列表', { exact: false })).toBeVisible()
    // 表格列头存在（字典相关列）
    const table = page.locator('.ant-table').first()
    await expect(table).toBeVisible()
    await expect(table.getByRole('columnheader').first()).toBeVisible()
  })
})
