import { test, expect } from '../fixtures/base'

/**
 * 冒烟：依次切换 7 个菜单，验证每个页面都能渲染（标题可见），且无 console error / 5xx。
 *
 * 前端用 state 切页（无路由库），因此通过点击左侧菜单导航，而非 URL 跳转。
 */
const MENUS = [
  { label: '商品管理', heading: '商品管理' },
  { label: '数据字典', heading: '数据字典' },
  { label: '配置中心', heading: '配置中心' },
  { label: '定时任务', heading: '定时任务' },
  { label: '异步任务', heading: '异步任务' },
  { label: '健康检查', heading: '健康检查' },
  { label: '文件上传', heading: '文件上传' },
]

test.describe('smoke · 菜单导航', () => {
  test('7 个页面均可渲染且无错误', async ({ page }) => {
    await page.goto('/')

    for (const m of MENUS) {
      await page.getByRole('menuitem', { name: m.label }).click()
      // 页面标题（Header 里的 Typography.Title）与菜单名一致，用 heading 断言可见。
      await expect(page.getByRole('heading', { name: m.heading, level: 4 })).toBeVisible()
    }
  })
})
