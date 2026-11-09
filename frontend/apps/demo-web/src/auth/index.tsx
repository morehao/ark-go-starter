import { useEffect, type ReactNode } from 'react'
import { Spin } from 'antd'
import { createAuthProvider, useAuthGuard } from '@morehao/ark-iam-react/auth'
import { setTokenProvider, setUnauthorizedHandler } from '@ark-go-starter/api'

/**
 * 门面包 AuthProvider：内部已含 `<BrowserRouter>` + react-oidc-context 的 `<AuthProvider>`，
 * 会先 `GET {runtimeConfigBase}/config.json` 拉运行时配置（成功/失败都落定后才渲染 children）。
 *
 * 关键约定：**外层不要再包 `<BrowserRouter>`**，否则 react-router 嵌套报错。
 * clientID / issuer 优先由 `/iam/config.json` 下发（useRuntimeConfig: true），
 * 也可用 `VITE_OIDC_CLIENT_ID` / `VITE_OIDC_ISSUER` 兜底；issuer 默认回退 `/oidc`。
 */
export const IAMAuthProvider = createAuthProvider({
  useRuntimeConfig: true,
  runtimeConfigBase: '/iam',
})

/**
 * 登录守卫 + 令牌桥接。必须渲染在 `IAMAuthProvider` 内部（依赖其 OIDC context）。
 *
 * 职责：
 * - `useAuthGuard()` 拿到 react-oidc-context 的 auth（含会话探活、401 会话过期自愈）；
 * - 把 access_token 注入本站 `@ark-go-starter/api`（门面包的 setUserProvider 只作用于它
 *   自己的 axios 实例，无法覆盖本站 api，故这里单独向本站 api 注册 tokenProvider）；
 * - 注册 401 未认证处理 → 跳 IAM 登录（signinRedirect）；
 * - 未登录且非加载中 → 触发 `signinRedirect()` 跳登录页。
 */
export function AuthGuard({ children }: { children: ReactNode }) {
  const auth = useAuthGuard()

  // 令牌注入：登录态变化时刷新本站 api 的 token 提供者
  useEffect(() => {
    setTokenProvider(() => auth.user?.access_token ?? null)
  }, [auth.user])

  // 401 会话过期 → 重新走 OIDC 登录（门面包 useAuthGuard 已同时 removeUser 清本地态）
  useEffect(() => {
    setUnauthorizedHandler(() => {
      void auth.removeUser().finally(() => void auth.signinRedirect())
    })
  }, [auth])

  // 未登录 → 跳转 OIDC 登录页
  useEffect(() => {
    if (!auth.isLoading && !auth.isAuthenticated) {
      void auth.signinRedirect()
    }
  }, [auth])

  // 卸载时解除注册，避免守卫撤下后仍用旧 token / 触发旧跳转
  useEffect(() => {
    return () => {
      setTokenProvider(null)
      setUnauthorizedHandler(null)
    }
  }, [])

  if (auth.isLoading || !auth.isAuthenticated) {
    return (
      <div
        style={{
          minHeight: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Spin size="large" tip="正在跳转登录…">
          <div style={{ width: 120, height: 40 }} />
        </Spin>
      </div>
    )
  }

  return <>{children}</>
}
