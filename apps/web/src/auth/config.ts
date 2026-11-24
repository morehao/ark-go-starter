/**
 * IAM 接入模式探测。
 *
 * 两态 mode：
 * - `disabled`（默认）：不接入 IAM，保持免登录现状，不挂 AuthProvider / AuthGuard；
 * - `rp`（Relying Party 对接外部 ark-iam）：前端直连外部 issuer，`useRuntimeConfig: false`，
 *   issuer 取 `VITE_OIDC_ISSUER`、clientID 取 `VITE_OIDC_CLIENT_ID`。
 *
 * detectIAMMode 判定顺序：
 * 1. `VITE_IAM_MODE` 显式指定优先（'disabled' | 'rp'），大小写/空格容错；
 * 2. 否则若设置了 `VITE_OIDC_ISSUER` 则视为 'rp'（外部 issuer 已就位）；
 * 3. 否则 'disabled'。
 */
export type IAMMode = 'disabled' | 'rp'

/** 读取 VITE_IAM_MODE 显式 mode；未设置或无法识别时返回 undefined。 */
function readExplicitMode(): IAMMode | undefined {
  const raw = import.meta.env.VITE_IAM_MODE
  if (raw === undefined || raw === null || raw === '') return undefined
  const v = String(raw).trim().toLowerCase()
  if (v === 'disabled' || v === 'rp') return v
  return undefined
}

/** 探测当前应采用的 IAM 接入模式（disabled | rp）。 */
export async function detectIAMMode(): Promise<IAMMode> {
  const explicit = readExplicitMode()
  if (explicit !== undefined) return explicit

  // 显式配置了外部 issuer → rp（直连远端）
  const externalIssuer = import.meta.env.VITE_OIDC_ISSUER
  if (externalIssuer && String(externalIssuer).trim() !== '') return 'rp'

  return 'disabled'
}

/**
 * 兼容旧 API：返回是否需要启用登录/守卫（rp）。
 * 新代码请直接用 detectIAMMode()。
 */
export async function detectIAMEnabled(): Promise<boolean> {
  const mode = await detectIAMMode()
  return mode !== 'disabled'
}
