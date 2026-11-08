import { cleanupE2EProducts } from './api'

/**
 * 全局清理：所有用例结束后，软删所有 e2e- 前缀商品，避免脏数据累积。
 * 即便有用例失败，teardown 仍会执行。
 */
export default async function globalTeardown(): Promise<void> {
  try {
    await cleanupE2EProducts()
  } catch (e) {
    // 清理失败不阻塞整体结果，仅打印提示。
    console.warn('[e2e] globalTeardown cleanup failed:', e)
  }
}
