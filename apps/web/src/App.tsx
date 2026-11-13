import { useEffect, useState } from 'react'
import { Layout, Menu, Typography } from 'antd'
import { tokens } from '@ark-go-starter/ui'
import ProductsPage from './pages/Products'
import DictsPage from './pages/Dicts'
import SettingsPage from './pages/Settings'
import CronPage from './pages/Cron'
import AsyncPage from './pages/Async'
import HealthPage from './pages/Health'
import FilesPage from './pages/Files'

const { Sider, Header, Content } = Layout

/** 菜单项 key 到页面组件的映射（不引入路由库，用 state 切换） */
const PAGES: Record<string, () => React.JSX.Element> = {
  products: ProductsPage,
  dicts: DictsPage,
  settings: SettingsPage,
  cron: CronPage,
  async: AsyncPage,
  health: HealthPage,
  files: FilesPage,
}

/** 侧栏分组：组名 + 组内菜单项（key / 标签 / 所属分组） */
const MENU_GROUPS: { title: string; items: { key: string; label: string }[] }[] = [
  {
    title: '数据管理',
    items: [
      { key: 'products', label: '商品管理' },
      { key: 'dicts', label: '数据字典' },
      { key: 'settings', label: '配置中心' },
    ],
  },
  {
    title: '运行时',
    items: [
      { key: 'cron', label: '定时任务' },
      { key: 'async', label: '异步任务' },
      { key: 'health', label: '健康检查' },
    ],
  },
  {
    title: '系统',
    items: [{ key: 'files', label: '文件上传' }],
  },
]

const MENU_LABELS: Record<string, string> = Object.fromEntries(
  MENU_GROUPS.flatMap((g) => g.items.map((i) => [i.key, i.label])),
)

/** 从 URL hash 读取当前页 key（如 #dicts），非法值回落到 products。 */
function readKeyFromHash(): string {
  const key = window.location.hash.replace(/^#/, '')
  return key && PAGES[key] ? key : 'products'
}

/**
 * 应用主布局。
 *
 * `headerRight` 是可选的页头右侧插槽：登录态（op/rp）由 `AuthedApp` 注入「头像 + 个人中心 /
 * 退出登录」入口；disabled 模式不传，页头保持原样（免登录现状）。
 */
export default function App({ headerRight }: { headerRight?: React.ReactNode } = {}) {
  const [activeKey, setActiveKey] = useState<string>(readKeyFromHash)
  const ActivePage = PAGES[activeKey] ?? ProductsPage

  // 切页时同步到 URL hash，支持刷新保持与直接分享定位
  useEffect(() => {
    window.location.hash = activeKey
  }, [activeKey])

  const menuItems = MENU_GROUPS.map((g) => ({
    type: 'group' as const,
    key: g.title,
    label: <span style={{ fontSize: 11, letterSpacing: '.06em', textTransform: 'uppercase' }}>{g.title}</span>,
    children: g.items.map((i) => ({ key: i.key, label: i.label })),
  }))

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        width={208}
        theme="dark"
        style={{
          background: tokens.sidebarBg,
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '0 20px',
            borderBottom: '1px solid rgba(255,255,255,0.08)',
          }}
        >
          <span
            style={{
              width: 28,
              height: 28,
              borderRadius: tokens.radiusMd,
              background: tokens.primary,
              display: 'grid',
              placeItems: 'center',
              color: '#ffffff',
              fontWeight: 800,
              fontSize: 15,
            }}
          >
            a
          </span>
          <div style={{ lineHeight: 1.15 }}>
            <div style={{ fontSize: tokens.fontSizeBody, fontWeight: 600, color: 'rgba(255,255,255,0.92)' }}>
              ark-go-starter
            </div>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)' }}>
              demo 控制台
            </div>
          </div>
        </div>
        <Menu
          mode="inline"
          theme="dark"
          selectedKeys={[activeKey]}
          items={menuItems}
          onClick={({ key }) => setActiveKey(key)}
          style={{
            background: tokens.sidebarBg,
            borderInlineEnd: 'none',
            padding: '8px 0',
          }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: tokens.headerBg,
            borderBottom: `1px solid ${tokens.border}`,
            paddingInline: 24,
          }}
        >
          <Typography.Text style={{ color: tokens.textSecondary, fontSize: tokens.fontSizeSmall }}>
            数据管理 /{' '}
            <Typography.Text strong style={{ color: tokens.text, fontSize: tokens.fontSizeSmall }}>
              {MENU_LABELS[activeKey] ?? ''}
            </Typography.Text>
          </Typography.Text>
          {headerRight}
        </Header>
        <Content style={{ padding: 24, background: tokens.bg }}>
          <ActivePage />
        </Content>
      </Layout>
    </Layout>
  )
}
