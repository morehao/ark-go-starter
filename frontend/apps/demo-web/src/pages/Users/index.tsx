import { useCallback, useEffect, useState } from 'react'
import { Table, type TableProps } from 'antd'
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
    { title: 'ID', dataIndex: 'userID', key: 'userID' },
    { title: '公司 ID', dataIndex: 'companyID', key: 'companyID' },
    { title: '部门 ID', dataIndex: 'departmentID', key: 'departmentID' },
    { title: '姓名', dataIndex: 'name', key: 'name' },
  ]

  return (
    <Table<UserPageListItem>
      rowKey="userID"
      columns={columns}
      dataSource={data}
      loading={loading}
      pagination={{
        current: page,
        pageSize,
        total,
        showTotal: (t) => `共 ${t} 条`,
        onChange: (p, ps) => {
          setPage(p)
          setPageSize(ps)
        },
      }}
    />
  )
}
