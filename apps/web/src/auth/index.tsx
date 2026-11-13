import { useEffect, type ReactNode } from 'react'
import { Spin } from 'antd'
import { createAuthProvider, useAuthGuard } from '@morehao/ark-iam-react/auth'
import { setTokenProvider, setUnauthorizedHandler } from '@ark-go-starter/api'
import type { IAMMode } from './config'

/**
 * 门面包 AuthProvider：内部已含 `<BrowserRouter>` + react-oidc-context 的 `<AuthProvider>`。
 *
 * 关键约定：**外层不要再包 `<BrowserRouter>`**，否则 react-router 嵌套报错。
 *
 * 两种接入模式（详见 ./config.ts 的 IAMMode）：
 * - `op`：内嵌本进程 IAM。`useRuntimeConfig: true` + `runtimeConfigBase: '/iam'`，
 *   clientID / issuer 优先由 `/iam/config.json` 下发（VITE_OIDC_CLIENT_ID / VITE_OIDC_ISSUER 兜底）。
 * - `rp`：对接外部 ark-iam。`useRuntimeConfig: false`，不走本站 `/iam`，
 *   直接使用 `VITE_OIDC_ISSUER` 作为 issuer、`VITE_OIDC_CLIENT_ID` 作为 clientID 直连远端 OP。
 */
function createIAMAuthProvider(mode: IAMMode) {
  // 登出回跳地址：门户包默认 `/login`，但本模板 SPA 没有 `/login` 路由（登录页在 `/iam/login`，
  // 由 OP 下发），且该地址必须逐字符命中客户端 post_logout_redirect_uris 白名单。这里统一改为
  // 回首页 `/`：登出后 AuthGuard 判定未登录 → 自动 signinRedirect 回登录页，体验等价且不会 400。
  // 对应后端 config.yaml 的 iam.post_logout_redirect_uris 需登记 `http://localhost:3000/`。
  const postLogoutRedirectPath = '/'
  if (mode === 'rp') {
    // 外部 issuer 绝对地址；config.ts 已保证 rp 时 VITE_OIDC_ISSUER 非空
    return createAuthProvider({
      useRuntimeConfig: false,
      issuer: import.meta.env.VITE_OIDC_ISSUER,
      clientID: import.meta.env.VITE_OIDC_CLIENT_ID,
      postLogoutRedirectPath,
    })
  }
  // op（默认走内嵌 /iam runtime config）
  return createAuthProvider({
    useRuntimeConfig: true,
    runtimeConfigBase: '/iam',
    postLogoutRedirectPath,
  })
}

/**
 * 按 mode 创建对应的门面包 AuthProvider。仅 op / rp 会挂载（disabled 不挂，见 main.tsx）。
 * 由 main.tsx 在渲染前根据 detectIAMMode() 结果调用。
 */
export function getIAMAuthProvider(mode: IAMMode) {
  return createIAMAuthProvider(mode)
}

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
        <Spin size="large" description="正在跳转登录…">
          <div style={{ width: 120, height: 40 }} />
        </Spin>
      </div>
    )
  }

  return <>{children}</>
}
