import React from 'react'
import ReactDOM from 'react-dom/client'
import { AppShell } from '@ark-go-starter/ui'
import App from './App'
import { detectIAMEnabled } from './auth/config'
import { AuthGuard, IAMAuthProvider } from './auth'

/**
 * 应用根节点。
 * - IAM 关闭：`<AppShell><App/></AppShell>`（保持现状免登录）；
 * - IAM 打开：在 AppShell 外层套 IAMAuthProvider（内含 BrowserRouter）+ AuthGuard（登录守卫）。
 */
function Root({ iamEnabled }: { iamEnabled: boolean }) {
  if (!iamEnabled) {
    return (
      <AppShell>
        <App />
      </AppShell>
    )
  }
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
  const iamEnabled = await detectIAMEnabled()
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <Root iamEnabled={iamEnabled} />
    </React.StrictMode>,
  )
}

void bootstrap()
