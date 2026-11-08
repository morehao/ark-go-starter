import { test, expect } from '../fixtures/base'

/**
 * 健康检查页：断言页面渲染出 PostgreSQL / Redis 两个依赖项，且状态为 ok。
 * 数据来自 GET /v1/demo/health，是真实后端 + 真实库/缓存的端到端信号。
 */
test.describe('健康检查', () => {
  test('展示 PostgreSQL / Redis 均 ok', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('menuitem', { name: '健康检查' }).click()
    await expect(page.getByRole('heading', { name: '健康检查', level: 5 })).toBeVisible()

    await expect(page.getByText('PostgreSQL', { exact: true })).toBeVisible()
    await expect(page.getByText('Redis', { exact: true })).toBeVisible()
    // 两个依赖项的状态标签都应为 ok（真实后端可用时才成立）。
    await expect(page.locator('.ant-tag', { hasText: 'ok' })).toHaveCount(2)
  })
})
