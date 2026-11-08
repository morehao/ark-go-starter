import { useState } from 'react'
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
const PAGES: Record<string, () => JSX.Element> = {
  products: ProductsPage,
  dicts: DictsPage,
  settings: SettingsPage,
  cron: CronPage,
  async: AsyncPage,
  health: HealthPage,
  files: FilesPage,
}

const MENU_ITEMS = [
  { key: 'products', label: '商品管理' },
  { key: 'dicts', label: '数据字典' },
  { key: 'settings', label: '配置中心' },
  { key: 'cron', label: '定时任务' },
  { key: 'async', label: '异步任务' },
  { key: 'health', label: '健康检查' },
  { key: 'files', label: '文件上传' },
]

export default function App() {
  const [activeKey, setActiveKey] = useState<string>('products')
  const ActivePage = PAGES[activeKey] ?? ProductsPage

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} style={{ background: tokens.sidebarBg }}>
        <div
          style={{
            height: 56,
            display: 'flex',
            alignItems: 'center',
            padding: '0 20px',
            color: tokens.headerBg,
            fontWeight: 600,
            fontSize: 15,
          }}
        >
          ark-go-starter
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[activeKey]}
          items={MENU_ITEMS}
          onClick={({ key }) => setActiveKey(key)}
          style={{ background: tokens.sidebarBg, borderInlineEnd: 'none' }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            display: 'flex',
            alignItems: 'center',
            borderBottom: `1px solid ${tokens.border}`,
          }}
        >
          <Typography.Title level={5} style={{ color: tokens.text, margin: 0 }}>
            {MENU_ITEMS.find((m) => m.key === activeKey)?.label ?? ''}
          </Typography.Title>
        </Header>
        <Content style={{ padding: 20 }}>
          <ActivePage />
        </Content>
      </Layout>
    </Layout>
  )
}
