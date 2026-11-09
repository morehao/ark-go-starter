/**
 * IAM 开关探测。
 *
 * 语义：
 * - iam 打开（后端挂载了内嵌 IAM，`/iam/config.json` 可达）→ 启用登录与守卫；
 * - iam 关闭（未挂载 IAM，`/iam/config.json` 不可达）→ 保持现状免登录，不挂守卫。
 *
 * 判定顺序：
 * 1. `VITE_IAM_ENABLE` 显式指定优先（'true'/'1' 打开、'false'/'0' 关闭），便于本地强制开关；
 * 2. 未显式指定则运行时探测 `GET {base}/config.json`（base 默认 `/iam`）。
 *
 * 与门面包 `createAuthProvider({ useRuntimeConfig: true, runtimeConfigBase: '/iam' })` 的
 * config.json 拉取口径一致：探测到可达才挂 AuthProvider，避免 IAM 关闭时反复报错。
 */
export const IAM_CONFIG_BASE = '/iam'

/** 读取 VITE_IAM_ENABLE 显式开关；未设置或无法识别时返回 undefined。 */
function readExplicitFlag(): boolean | undefined {
  const raw = import.meta.env.VITE_IAM_ENABLE
  if (raw === undefined || raw === null || raw === '') return undefined
  const v = String(raw).toLowerCase()
  if (v === 'true' || v === '1') return true
  if (v === 'false' || v === '0') return false
  return undefined
}

/** 运行时探测 IAM 是否可用：返回 true 表示应启用登录/守卫。 */
export async function detectIAMEnabled(): Promise<boolean> {
  const explicit = readExplicitFlag()
  if (explicit !== undefined) return explicit
  try {
    const res = await fetch(`${IAM_CONFIG_BASE}/config.json`, {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    })
    return res.ok
  } catch {
    return false
  }
}
