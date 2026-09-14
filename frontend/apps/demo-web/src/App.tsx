import { Layout, Typography } from 'antd'
import UsersPage from './pages/Users'

export default function App() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Header>
        <Typography.Title level={4} style={{ color: '#fff', lineHeight: '64px', margin: 0 }}>
          ark-go-starter demo
        </Typography.Title>
      </Layout.Header>
      <Layout.Content style={{ padding: 24 }}>
        <UsersPage />
      </Layout.Content>
    </Layout>
  )
}
