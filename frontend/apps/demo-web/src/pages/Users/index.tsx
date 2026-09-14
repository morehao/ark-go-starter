import { useCallback, useEffect, useState } from 'react'
import { Button, Table, type TableProps } from 'antd'
import { PageContainer } from '@ark-go-starter/ui'
import { getUsers } from '@ark-go-starter/api'
import type { UserPageListItem } from '@ark-go-starter/types'

export default function UsersPage() {
  const [data, setData] = useState<UserPageListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getUsers({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize])

  useEffect(() => {
    load()
  }, [load])

  const columns: TableProps<UserPageListItem>['columns'] = [
    { title: 'ID', dataIndex: 'userID', key: 'userID', width: 130 },
    { title: '公司 ID', dataIndex: 'companyID', key: 'companyID', width: 120 },
    { title: '部门 ID', dataIndex: 'departmentID', key: 'departmentID', width: 120 },
    { title: '姓名', dataIndex: 'name', key: 'name', width: 180 },
  ]

  return (
    <PageContainer
      title="用户管理"
      description="演示后端分页接口，数据来自 GET /v1/demo/users"
      extra={<Button onClick={load}>刷新</Button>}
    >
      <Table<UserPageListItem>
        rowKey="userID"
        tableLayout="fixed"
        scroll={{ x: 550 }}
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />
    </PageContainer>
  )
}
