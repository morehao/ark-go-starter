import { test, expect } from '../fixtures/base'

/**
 * 健康检查页：断言页面渲染出 PostgreSQL / Redis 两个依赖项，且状态为 ok。
 * 数据来自 GET /v1/demo/health，是真实后端 + 真实库/缓存的端到端信号。
 *
 * 失效态（停库/停 Redis）需要破坏共享环境，e2e 不主动制造；改为断言「刷新按钮会重新打接口
 * 且仍拿到 code:0」，覆盖前端刷新逻辑与后端健康检查链路。
 */
test.describe('健康检查', () => {
  test('展示 PostgreSQL / Redis 均 ok', async ({ page }) => {
    await page.goto('/')
    const [healthResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/v1/demo/health') && r.request().method() === 'GET',
      ),
      page.getByRole('menuitem', { name: '健康检查' }).click(),
    ])
    const body = (await healthResp.json()) as {
      code: number
      data: { postgres: string; redis: string }
    }
    expect(body.code).toBe(0)
    expect(body.data.postgres).toBe('ok')
    expect(body.data.redis).toBe('ok')

    await expect(page.getByRole('heading', { name: '健康检查', level: 4 })).toBeVisible()
    await expect(page.getByText('PostgreSQL', { exact: true })).toBeVisible()
    await expect(page.getByText('Redis', { exact: true })).toBeVisible()
    // 两个依赖项的状态标签都应为 ok（真实后端可用时才成立）
    await expect(page.locator('.ant-tag-success', { hasText: 'ok' })).toHaveCount(2)
  })

  test('刷新：重新请求健康检查接口并保持两依赖 ok', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('menuitem', { name: '健康检查' }).click()
    await expect(page.getByRole('heading', { name: '健康检查', level: 4 })).toBeVisible()
    await expect(page.locator('.ant-tag', { hasText: 'ok' })).toHaveCount(2)

    const [refreshResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/v1/demo/health') && r.request().method() === 'GET',
      ),
      page.getByRole('button', { name: /刷\s*新/ }).click(),
    ])
    expect(((await refreshResp.json()) as { code: number }).code).toBe(0)
    await expect(page.locator('.ant-tag', { hasText: 'ok' })).toHaveCount(2)
  })
})
