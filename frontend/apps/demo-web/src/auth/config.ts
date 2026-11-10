/**
 * IAM 接入模式探测。
 *
 * 三态 mode：
 * - `disabled`（默认）：不接入 IAM，保持免登录现状，不挂 AuthProvider / AuthGuard；
 * - `op`（OpenID Provider 内嵌本进程）：后端在本进程挂了内嵌 IAM，前端走同源 `/iam`，
 *   由 `GET /iam/config.json` 下发 issuer/clientId，`createAuthProvider` 用 `runtimeConfigBase: '/iam'`；
 * - `rp`（Relying Party 对接外部 ark-iam）：前端直连外部 issuer，`useRuntimeConfig: false`，
 *   issuer 取 `VITE_OIDC_ISSUER`、clientID 取 `VITE_OIDC_CLIENT_ID`，不依赖本站 `/iam`。
 *
 * detectIAMMode 判定顺序：
 * 1. `VITE_IAM_MODE` 显式指定优先（'disabled' | 'op' | 'rp'），大小写/空格容错；
 * 2. 兼容旧变量 `VITE_IAM_ENABLE`：true/1 → 'op'，false/0 → 'disabled'；
 * 3. 未显式指定：若设置了 `VITE_OIDC_ISSUER` 则视为 'rp'（外部 issuer 已就位）；
 * 4. 否则运行时探测 `GET /iam/config.json` 是否可达，可达 → 'op'；
 * 5. 都不满足 → 'disabled'。
 */
export type IAMMode = 'disabled' | 'op' | 'rp'

/** 内嵌 IAM（op）运行时配置基址。 */
export const IAM_CONFIG_BASE = '/iam'

/** 读取 VITE_IAM_MODE 显式 mode；未设置或无法识别时返回 undefined。 */
function readExplicitMode(): IAMMode | undefined {
  const raw = import.meta.env.VITE_IAM_MODE
  if (raw === undefined || raw === null || raw === '') return undefined
  const v = String(raw).trim().toLowerCase()
  if (v === 'disabled' || v === 'op' || v === 'rp') return v
  return undefined
}

/**
 * 兼容旧开关 VITE_IAM_ENABLE：映射为 mode。
 * 'true'/'1' → 'op'；'false'/'0' → 'disabled'；其余返回 undefined。
 */
function readLegacyMode(): IAMMode | undefined {
  const raw = import.meta.env.VITE_IAM_ENABLE
  if (raw === undefined || raw === null || raw === '') return undefined
  const v = String(raw).trim().toLowerCase()
  if (v === 'true' || v === '1') return 'op'
  if (v === 'false' || v === '0') return 'disabled'
  return undefined
}

/** 探测本进程内嵌 IAM（op）是否可用：`GET /iam/config.json` 可达即视为 op。 */
async function probeEmbeddedOP(): Promise<boolean> {
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

/** 探测当前应采用的 IAM 接入模式（disabled | op | rp）。 */
export async function detectIAMMode(): Promise<IAMMode> {
  const explicit = readExplicitMode()
  if (explicit !== undefined) return explicit

  const legacy = readLegacyMode()
  if (legacy !== undefined) return legacy

  // 显式配置了外部 issuer → rp（直连远端，无需本站 /iam）
  const externalIssuer = import.meta.env.VITE_OIDC_ISSUER
  if (externalIssuer && String(externalIssuer).trim() !== '') return 'rp'

  // 否则探测内嵌 OP 是否可达
  if (await probeEmbeddedOP()) return 'op'

  return 'disabled'
}

/**
 * 兼容旧 API：返回是否需要启用登录/守卫（op 或 rp）。
 * 新代码请直接用 detectIAMMode()。
 */
export async function detectIAMEnabled(): Promise<boolean> {
  const mode = await detectIAMMode()
  return mode !== 'disabled'
}
