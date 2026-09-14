import { Layout, Typography } from 'antd'
import { tokens } from '@ark-go-starter/ui'
import UsersPage from './pages/Users'

export default function App() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Header style={{ display: 'flex', alignItems: 'center' }}>
        <Typography.Title level={4} style={{ color: tokens.text, margin: 0 }}>
          ark-go-starter demo
        </Typography.Title>
      </Layout.Header>
      <Layout.Content style={{ padding: 20 }}>
        <UsersPage />
      </Layout.Content>
    </Layout>
  )
}
