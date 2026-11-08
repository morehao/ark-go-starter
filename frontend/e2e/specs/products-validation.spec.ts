import { test, expect } from '../fixtures/base'

/**
 * 商品表单校验：名称为空时提交被拦截，显示「请输入商品名称」，弹窗保持打开。
 */
test.describe('商品管理 · 表单校验', () => {
  test('名称必填：空名称提交被拦截', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: '商品管理', level: 5 })).toBeVisible()

    await page.getByTestId('product-create-btn').click()
    const modal = page.getByRole('dialog')
    await expect(modal.getByText('新建商品')).toBeVisible()

    // 不填名称，直接提交
    await modal.getByRole('button', { name: /确\s*定|OK/i }).click()

    // 校验提示出现，弹窗仍打开
    await expect(modal.getByText('请输入商品名称')).toBeVisible()
    await expect(modal).toBeVisible()

    // 关闭弹窗，避免影响其它用例
    await modal.getByRole('button', { name: /取\s*消|Cancel/i }).click()
    await expect(modal).toBeHidden()
  })
})
