import { useCallback, useEffect, useState } from 'react'
import { Button, Descriptions, Tag, Tooltip, App, type DescriptionsProps } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { PageContainer } from '@ark-go-starter/ui'
import { getHealth } from '@ark-go-starter/api'
import type { DbCheck } from '@ark-go-starter/types'

/** 'ok'→success 色，其它→error 色 */
function HealthTag({ value }: { value?: string }) {
  if (!value) return <Tag color="default">未知</Tag>
  return value === 'ok' ? (
    <Tag color="success">ok</Tag>
  ) : (
    <Tag color="error">{value}</Tag>
  )
}

export default function HealthPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<DbCheck | null>(null)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getHealth()
      setData(res)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '健康检查失败')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [message])

  useEffect(() => {
    load()
  }, [load])

  const items: DescriptionsProps['items'] = [
    {
      key: 'postgres',
      label: 'PostgreSQL',
      children: <HealthTag value={data?.postgres} />,
    },
    {
      key: 'redis',
      label: 'Redis',
      children: <HealthTag value={data?.redis} />,
    },
  ]

  return (
    <PageContainer
      title="健康检查"
      description="检查后端依赖连通性，数据来自 GET /v1/demo/health"
      extra={
        <Tooltip title="刷新">
          <Button icon={<ReloadOutlined />} aria-label="刷新" loading={loading} onClick={load} />
        </Tooltip>
      }
    >
      <Descriptions bordered size="small" column={1} items={items} />
    </PageContainer>
  )
}
