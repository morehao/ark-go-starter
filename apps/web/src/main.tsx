import React from 'react'
import ReactDOM from 'react-dom/client'
import { AppShell } from '@ark-go-starter/ui'
import App from './App'
import { detectIAMMode, type IAMMode } from './auth/config'
import { getIAMAuthProvider } from './auth'
import { AuthedApp } from './auth/AuthedApp'

/**
 * 应用根节点（两态 mode）。
 * - `disabled`：AppShell 直接包裹 App（免登录现状，无个人中心/登出）；
 * - `rp`：外部 OP 自带登录页，全部路径套 IAMAuthProvider + AuthedApp
 *   （AuthedApp 内含 AuthGuard，并向页面注入「个人中心 + 登出」页头入口）。
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
      <AuthedApp
        render={({ headerRight }) => (
          <AppShell>
            <App headerRight={headerRight} />
          </AppShell>
        )}
      />
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
