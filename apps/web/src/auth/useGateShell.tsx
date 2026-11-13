import { useCallback, useState } from 'react'
import { message } from 'antd'
import { useAuthGuard, useLogout } from '@morehao/ark-iam-react/auth'
import { ProfileCenter } from '@morehao/ark-iam-react'
import type { ReactNode } from 'react'
import { ProfileTokenBridge } from './useGateShellBridge'

/**
 * 认证态「外壳」：把「个人中心 + 登出」能力打包给布局使用。
 *
 * 复用 `@morehao/ark-iam-react`（与 ark-iam 的 platform-admin-web 同一套底层）：
 * - 个人中心 = 包内 `ProfileCenter`（基本信息 / 修改密码 / 会话管理）；
 * - 登出 = 包内 `useLogout()`（撤销会话 → 跳 OP end_session 做 SSO 登出 → 清本地态）。
 *
 * `@ark-go-starter/ui`（含 `AppShell`）刻意不依赖 IAM 包，避免 disabled 模式也打进 OIDC
 * 依赖；而 `ProfileCenter` / `useLogout` 都依赖 OIDC context，必须在 `IAMAuthProvider` 内
 * 渲染。故登录态外壳落在本仓库前端：本钩子在 `IAMAuthProvider` 内调用，返回页头入口与浮层。
 */
export interface GateShell {
  /** 打开个人中心（基本信息 / 修改密码 / 会话管理）。 */
  openProfile: () => void
  /** 登出：撤销会话并走 OP 的 end_session 完成 SSO 登出。 */
  logout: () => Promise<void>
  /** 个人中心浮层节点（需渲染到组件树中才会出现）。 */
  profileNode: ReactNode
}

/** 在登录态下获取「个人中心 + 登出」能力（须位于 `IAMAuthProvider` 内）。 */
export function useGateShell(): GateShell {
  const auth = useAuthGuard()
  // 门面包 useLogout：logoutAllAPI（凭 refresh_token 撤销全部会话）→ signoutRedirect
  // （跳 OP end_session 做 SSO 登出）→ removeUser 清本地态；登出后 OP 按配置回跳前端。
  const logout = useLogout()
  const [profileOpen, setProfileOpen] = useState(false)

  const openProfile = useCallback(() => setProfileOpen(true), [])
  const closeProfile = useCallback(() => setProfileOpen(false), [])

  const handleLogout = useCallback(async () => {
    try {
      await logout()
    } catch {
      message.error('登出失败，请重试')
    }
  }, [logout])

  return {
    openProfile,
    logout: handleLogout,
    // ProfileTokenBridge 负责把当前 access_token 注入门面包 axios 单例（ProfileCenter 用），
    // 再渲染 ProfileCenter；两者都用同一份令牌快照，避免首帧 401。
    profileNode: (
      <ProfileTokenBridge accessToken={auth.user?.access_token ?? null}>
        <ProfileCenter open={profileOpen} onClose={closeProfile} />
      </ProfileTokenBridge>
    ),
  }
}
