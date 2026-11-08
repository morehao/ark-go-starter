import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright e2e 配置。
 *
 * 运行前提：后端已在 http://localhost:8099 提供服务（由 Makefile / CI job 负责拉起），
 * 前端通过 vite 的 /v1 代理打通到后端，形成真正的端到端链路。
 *
 * 前端服务启动方式（二选一）：
 *   - 开发模式（本地默认）：pnpm dev，端口 3000，复用已运行实例。
 *   - 生产预览（CI 默认）：先 build 再 preview，端口 4173。
 * 通过 E2E_WEB_MODE=dev|preview 切换，默认 dev。
 */
const WEB_MODE = process.env.E2E_WEB_MODE === 'preview' ? 'preview' : 'dev'
const WEB_PORT = WEB_MODE === 'preview' ? 4173 : 3000
const WEB_CMD =
  WEB_MODE === 'preview'
    ? 'pnpm --filter @ark-go-starter/demo-web build && pnpm --filter @ark-go-starter/demo-web preview --port 4173 --strictPort'
    : 'pnpm --filter @ark-go-starter/demo-web dev'

export default defineConfig({
  testDir: './specs',
  globalTeardown: './support/teardown.ts',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  // 共享真实后端数据库，CI 串行执行避免数据竞争；本地可按需调高。
  workers: process.env.CI ? 1 : undefined,
  timeout: 30_000,
  expect: { timeout: 10_000 },
  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never' }]]
    : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: `http://localhost:${WEB_PORT}`,
    locale: 'zh-CN',
    timezoneId: 'Asia/Shanghai',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    // 预留跨浏览器矩阵（默认关闭）：
    // { name: 'webkit', use: { ...devices['Desktop Safari'] } },
    // { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
  ],
  webServer: {
    command: WEB_CMD,
    url: `http://localhost:${WEB_PORT}`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
})
