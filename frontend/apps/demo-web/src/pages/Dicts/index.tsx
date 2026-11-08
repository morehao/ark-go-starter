import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Drawer,
  Popconfirm,
  Space,
  Table,
  Tag,
  App,
  type TableProps,
} from 'antd'
import { PageContainer } from '@ark-go-starter/ui'
import {
  deleteDictType,
  getDictItems,
  getDictTypes,
  updateDictTypeStatus,
} from '@ark-go-starter/api'
import type { DictItem, DictType } from '@ark-go-starter/types'

/** unix 秒转本地时间字符串，空值显示 '-' */
function fmtTime(v?: number | null): string {
  if (v === undefined || v === null) return '-'
  return new Date(v * 1000).toLocaleString()
}

/** 状态 Tag 语义色：enabled→success，其余→default */
function StatusTag({ status }: { status: string }) {
  return status === 'enabled' ? (
    <Tag color="success">启用</Tag>
  ) : (
    <Tag color="default">禁用</Tag>
  )
}

export default function DictsPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<DictType[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTitle, setDrawerTitle] = useState('')
  const [items, setItems] = useState<DictItem[]>([])
  const [itemsLoading, setItemsLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getDictTypes({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '加载字典类型失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, message])

  useEffect(() => {
    load()
  }, [load])

  const toggleStatus = async (record: DictType) => {
    const next = record.status === 'enabled' ? 'disabled' : 'enabled'
    try {
      await updateDictTypeStatus(record.id, next)
      message.success('状态已更新')
      load()
    } catch (e) {
      message.error(e instanceof Error ? e.message : '状态更新失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteDictType(id)
      message.success('删除成功')
      load()
    } catch (e) {
      message.error(e instanceof Error ? e.message : '删除失败')
    }
  }

  const openItems = async (record: DictType) => {
    setDrawerTitle(`字典项 · ${record.name || record.code}`)
    setDrawerOpen(true)
    setItemsLoading(true)
    try {
      const res = await getDictItems(record.code, { includeDisabled: true })
      setItems(res.list ?? [])
    } catch (e) {
      message.error(e instanceof Error ? e.message : '加载字典项失败')
    } finally {
      setItemsLoading(false)
    }
  }

  const columns: TableProps<DictType>['columns'] = [
    { title: '编码', dataIndex: 'code', key: 'code', width: 160, ellipsis: true },
    { title: '名称', dataIndex: 'name', key: 'name', width: 160, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <StatusTag status={s} />,
    },
    { title: '说明', dataIndex: 'description', key: 'description', width: 220, ellipsis: true },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 190,
      render: (v: number) => fmtTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      render: (_, record) => (
        <Space size="middle">
          <Button type="link" size="small" onClick={() => openItems(record)}>
            查看项
          </Button>
          <Button type="link" size="small" onClick={() => toggleStatus(record)}>
            {record.status === 'enabled' ? '禁用' : '启用'}
          </Button>
          <Popconfirm
            title="确认删除该字典类型？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const itemColumns: TableProps<DictItem>['columns'] = [
    { title: '码值', dataIndex: 'value', key: 'value', width: 140, ellipsis: true },
    { title: '展示文案', dataIndex: 'label', key: 'label', width: 160, ellipsis: true },
    { title: '层级', dataIndex: 'level', key: 'level', width: 80 },
    { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <StatusTag status={s} />,
    },
    { title: '说明', dataIndex: 'description', key: 'description', width: 180, ellipsis: true },
  ]

  return (
    <PageContainer
      title="数据字典"
      description="字典类型分页列表，可查看字典项、启停与删除，数据来自 /v1/dict-types"
      extra={<Button onClick={load}>刷新</Button>}
    >
      <Table<DictType>
        rowKey="id"
        tableLayout="fixed"
        scroll={{ x: 1050 }}
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

      <Drawer
        title={drawerTitle}
        width={820}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      >
        <Table<DictItem>
          rowKey="id"
          tableLayout="fixed"
          scroll={{ x: 740 }}
          columns={itemColumns}
          dataSource={items}
          loading={itemsLoading}
          pagination={false}
        />
      </Drawer>
    </PageContainer>
  )
}
