import React from 'react'
import ReactDOM from 'react-dom/client'
import { AppShell } from '@ark-go-starter/ui'
import App from './App'
import { detectIAMMode, type IAMMode } from './auth/config'
import { AuthGuard, getIAMAuthProvider } from './auth'

/**
 * 应用根节点（三态 mode）。
 * - `disabled`：AppShell 直接包裹 App（免登录现状）；
 * - `op` / `rp`：外层套对应 mode 的 IAMAuthProvider（内含 BrowserRouter）+ AuthGuard（登录守卫）。
 */
function Root({ mode }: { mode: IAMMode }) {
  if (mode === 'disabled') {
    return (
      <AppShell>
        <App />
      </AppShell>
    )
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
