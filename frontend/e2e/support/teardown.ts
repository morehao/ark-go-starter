import { cleanupE2EProducts } from './api'

/**
 * 全局清理：所有用例结束后，软删所有 e2e- 前缀商品，避免脏数据累积。
 * 即便有用例失败，teardown 仍会执行。
 */
export default async function globalTeardown(): Promise<void> {
  // IAM 开启时业务接口走 OIDC fail-closed（免登录会 401），此免登录清理无意义且会产生噪音，
  // 且 IAM 场景不通过免登录接口造 e2e- 商品，故直接跳过。
  if (process.env.E2E_IAM === '1') return
  try {
    await cleanupE2EProducts()
  } catch (e) {
    // 清理失败不阻塞整体结果，仅打印提示。
    console.warn('[e2e] globalTeardown cleanup failed:', e)
  }
}
