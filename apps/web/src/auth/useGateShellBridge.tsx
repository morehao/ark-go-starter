import { useEffect, type ReactNode } from 'react'
import { setUserProvider } from '@morehao/ark-iam-react'

/**
 * 令牌桥接：把当前 OIDC access_token 注入门面包（`@morehao/ark-iam-react`）内部的 axios 单例。
 *
 * 背景：包内 `ProfileCenter` 用**它自己的** axios 单例（`baseURL=/v1`）请求 `/auth/me`、
 * `/auth/me/sessions`、`/auth/me/changePassword`，其 `Authorization` 头来自全局
 * `userProvider`。该 provider 原由门面包 `useAuthGuard` 设置；本仓库 `AuthGuard` 复用它，
 * 但仍在此显式再设一次，保证 ProfileCenter 的请求一定带 Bearer（否则一律 401）。
 *
 * 单独成组件是为了把「改外部单例」收敛为一个副作用，避免渲染期直接改（React 规范 +
 * 本仓 React Compiler 规则）。
 */
export function ProfileTokenBridge({
  accessToken,
  children,
}: {
  accessToken: string | null
  children: ReactNode
}) {
  useEffect(() => {
    setUserProvider(() => (accessToken ? { access_token: accessToken } : null))
  }, [accessToken])

  return <>{children}</>
}
