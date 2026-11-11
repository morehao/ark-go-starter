import React from 'react'
import ReactDOM from 'react-dom/client'
import { AppShell } from '@ark-go-starter/ui'
import { LoginPage } from '@morehao/ark-iam-react/login'
import App from './App'
import { detectIAMMode, type IAMMode } from './auth/config'
import { AuthGuard, getIAMAuthProvider } from './auth'

/**
 * OP 内嵌账密登录页：`GET /iam/login?authRequestID=…` 由 OP（后端）下发，浏览器导航到本 SPA。
 *
 * 该页面**不套** AuthProvider / AuthGuard（否则会被守卫再次重定向，形成回环），
 * 只依赖 antd `ConfigProvider`（由 `AppShell` 提供主题/语言包）渲染门面包的 `LoginPage`。
 * `basePath` 默认从 `window.location.pathname`（`/iam/login` → `/iam`）推断，与后端挂载前缀一致。
 */
function IAMLoginPage() {
  return (
    <AppShell>
      <LoginPage />
    </AppShell>
  )
}

/**
 * 应用根节点（三态 mode）。
 * - `disabled`：AppShell 直接包裹 App（免登录现状）；
 * - `op`：`/iam/login` 渲染 OP 账密页（不套守卫）；其余路径套 IAMAuthProvider + AuthGuard；
 * - `rp`：外部 OP 自带登录页，不在本站渲染账密页，全部路径套 IAMAuthProvider + AuthGuard。
 */
function Root({ mode }: { mode: IAMMode }) {
  if (mode === 'disabled') {
    return (
      <AppShell>
        <App />
      </AppShell>
    )
  }
  // op 模式下的 OP 账密页：仅 GET /iam/login 导航命中，独立渲染（不套 Provider/Guard）。
  if (mode === 'op' && window.location.pathname === '/iam/login') {
    return <IAMLoginPage />
  }
  const IAMAuthProvider = getIAMAuthProvider(mode)
  return (
    <IAMAuthProvider>
      <AuthGuard>
        <AppShell>
          <App />
        </AppShell>
      </AuthGuard>
    </IAMAuthProvider>
  )
}

async function bootstrap() {
  const mode = await detectIAMMode()
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <Root mode={mode} />
    </React.StrictMode>,
  )
}

void bootstrap()
