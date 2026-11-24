import { cleanupE2EDicts, cleanupE2EProducts, cleanupE2ESettings } from './api'

/**
 * 全局清理：所有用例结束后，按 e2e- 前缀清理字典类型/配置项/商品，避免脏数据累积。
 * 即便有用例失败，teardown 仍会执行；单项失败只打警告，不阻塞整体结果。
 *
 * 说明：
 * - 文件（files）没有列表接口，无法按前缀兜底清理，统一由 files spec 的 afterEach 按 id 删；
 * - cron/async 任务由后端代码注册、不由 e2e 创建，这里不需要清理。
 */
export default async function globalTeardown(): Promise<void> {
  // 旧 op 形态下 IAM=ON 时业务接口走 OIDC fail-closed（免登录会 401），此免登录清理无意义，保留跳过。
  // 注：op 已下线，iam-login.spec.ts 现整组 skip；默认 disabled 免登录下 E2E_IAM 恒不为 1，正常清理。
  if (process.env.E2E_IAM === '1') return

  for (const [name, fn] of [
    ['products', cleanupE2EProducts],
    ['dicts', cleanupE2EDicts],
    ['settings', cleanupE2ESettings],
  ] as const) {
    try {
      await fn()
    } catch (e) {
      // 清理失败不阻塞整体结果，仅打印提示。
      console.warn(`[e2e] globalTeardown cleanup ${name} failed:`, e)
    }
  }
}
