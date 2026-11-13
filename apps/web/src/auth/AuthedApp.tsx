import { useEffect, useState, type ReactNode } from 'react'
import { Avatar, Dropdown } from 'antd'
import { LogoutOutlined, UserOutlined } from '@ant-design/icons'
import { getUserinfo } from '@morehao/ark-iam-react'
import { tokens } from '@ark-go-starter/ui'
import { AuthGuard } from './index'
import { useGateShell } from './useGateShell'

/**
 * 登录态应用外壳：在 `AuthGuard` 内部把「个人中心 + 登出」注入到页面树。
 *
 * `render` 收到 `headerRight`（页头右侧的用户入口），由页面把它放进 Header；
 * 同时渲染 `profileNode`（个人中心浮层）。这样个人中心/登出只在 op / rp 登录态出现，
 * disabled 模式（免登录现状）不受影响。
 */
export function AuthedApp({ render }: { render: (chrome: AuthedChrome) => ReactNode }) {
  return (
    <AuthGuard>
      <AuthedInner render={render} />
    </AuthGuard>
  )
}

export interface AuthedChrome {
  /** 页头右侧用户入口（头像 + 下拉：个人中心 / 退出登录）。 */
  headerRight: ReactNode
}

function AuthedInner({ render }: { render: (chrome: AuthedChrome) => ReactNode }) {
  const { openProfile, logout, profileNode } = useGateShell()
  const [displayName, setDisplayName] = useState('')

  // 拉一次当前身份用于头像/名称展示；失败则回落到「用户」占位，不阻断页面。
  useEffect(() => {
    let cancelled = false
    getUserinfo()
      .then((info) => {
        if (cancelled) return
        setDisplayName(info.personInfo?.name || info.userInfo?.name || info.personInfo?.personID || '用户')
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [])

  const headerRight = (
    <Dropdown
      trigger={['click']}
      menu={{
        items: [
          {
            key: 'profile',
            icon: <UserOutlined />,
            label: '个人中心',
            onClick: openProfile,
          },
          { type: 'divider' },
          {
            key: 'logout',
            icon: <LogoutOutlined />,
            danger: true,
            label: '退出登录',
            onClick: () => void logout(),
          },
        ],
      }}
    >
      <button
        type="button"
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 8,
          height: 40,
          padding: '0 10px',
          border: 'none',
          background: 'transparent',
          cursor: 'pointer',
          borderRadius: tokens.radiusMd,
        }}
      >
        <Avatar size={28} style={{ background: tokens.primary, fontSize: 13 }}>
          {displayName ? displayName.charAt(0).toUpperCase() : <UserOutlined />}
        </Avatar>
        <span style={{ color: tokens.text, fontSize: tokens.fontSizeSmall, maxWidth: 120 }}>
          {displayName || '用户'}
        </span>
      </button>
    </Dropdown>
  )

  return (
    <>
      {render({ headerRight })}
      {profileNode}
    </>
  )
}
