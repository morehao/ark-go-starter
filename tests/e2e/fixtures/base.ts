import { test as base, expect } from '@playwright/test'

/**
 * 自定义 test 基座。
 *
 * 收集每个用例期间的运行时错误，作为附加断言：
 *   - pageerror（未捕获 JS 异常）→ 一律视为失败（前端崩溃的最强信号）；
 *   - HTTP 5xx → 一律视为失败（后端故障）。
 *
 * 不把 4xx 与 console.error 视为失败：demo 应用中部分可选模块（dict/settings/cron/async）
 * 在特定运行方式下未注册会返回 404，antd 也存在已知的弃用警告；这些属于已知噪音，
 * 若作为硬门会持续误报。需要严格校验某页时，在具体 spec 内单独断言其接口状态码。
 */
export const test = base.extend<{ runtimeGuard: void }>({
  runtimeGuard: [
    async ({ page }, use) => {
      const fatal: string[] = []
      page.on('pageerror', (err) => fatal.push(`pageerror: ${err.message}`))
      page.on('response', (res) => {
        if (res.status() >= 500) fatal.push(`http ${res.status()}: ${res.url()}`)
      })

      await use()

      expect(fatal, `页面运行时存在致命错误：\n${fatal.join('\n')}`).toEqual([])
    },
    { auto: true },
  ],
})

export { expect }
