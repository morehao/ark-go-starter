import { useCallback, useEffect, useState } from 'react'
import { Button, Drawer, Table, Tag, App, type TableProps } from 'antd'
import { PageContainer } from '@ark-go-starter/ui'
import { getCronTaskRuns, getCronTasks, updateCronTaskStatus } from '@ark-go-starter/api'
import type { CronTask, CronTaskRun } from '@ark-go-starter/types'

/** unix 秒转本地时间字符串，空值显示 '-' */
function fmtTime(v?: number | null): string {
  if (v === undefined || v === null) return '-'
  return new Date(v * 1000).toLocaleString()
}

/** 状态 Tag 语义色：enabled→success，disabled→default，其它→warning */
function StatusTag({ status }: { status: string }) {
  if (status === 'enabled') return <Tag color="success">启用</Tag>
  if (status === 'disabled') return <Tag color="default">禁用</Tag>
  return <Tag color="warning">{status || '-'}</Tag>
}

export default function CronPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<CronTask[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTitle, setDrawerTitle] = useState('')
  const [runs, setRuns] = useState<CronTaskRun[]>([])
  const [runsLoading, setRunsLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getCronTasks({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '加载定时任务失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, message])

  useEffect(() => {
    load()
  }, [load])

  const toggleStatus = async (record: CronTask) => {
    const next = record.status === 'enabled' ? 'disabled' : 'enabled'
    try {
      await updateCronTaskStatus(record.id, next)
      message.success('状态已更新')
      load()
    } catch (e) {
      message.error(e instanceof Error ? e.message : '状态更新失败')
    }
  }

  const openRuns = async (record: CronTask) => {
    setDrawerTitle(`运行记录 · ${record.name || record.id}`)
    setDrawerOpen(true)
    setRunsLoading(true)
    try {
      const res = await getCronTaskRuns(record.id)
      setRuns(res.list ?? [])
    } catch (e) {
      message.error(e instanceof Error ? e.message : '加载运行记录失败')
    } finally {
      setRunsLoading(false)
    }
  }

  const columns: TableProps<CronTask>['columns'] = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 180, ellipsis: true },
    { title: '任务类型', dataIndex: 'task_type', key: 'task_type', width: 140, ellipsis: true },
    { title: 'Cron 表达式', dataIndex: 'spec', key: 'spec', width: 160, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <StatusTag status={s} />,
    },
    {
      title: '下次运行时间',
      dataIndex: 'next_run_at',
      key: 'next_run_at',
      width: 190,
      render: (v: number | null | undefined) => fmtTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, record) => (
        <span style={{ display: 'flex', gap: 16 }}>
          <Button type="link" size="small" onClick={() => openRuns(record)}>
            运行记录
          </Button>
          <Button type="link" size="small" onClick={() => toggleStatus(record)}>
            {record.status === 'enabled' ? '禁用' : '启用'}
          </Button>
        </span>
      ),
    },
  ]

  const runColumns: TableProps<CronTaskRun>['columns'] = [
    { title: '开始时间', dataIndex: 'start_at', key: 'start_at', width: 190, render: (v: number) => fmtTime(v) },
    { title: '结束时间', dataIndex: 'end_at', key: 'end_at', width: 190, render: (v: number | null | undefined) => fmtTime(v) },
    { title: '耗时(ms)', dataIndex: 'duration_ms', key: 'duration_ms', width: 110 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <StatusTag status={s} />,
    },
    { title: '错误信息', dataIndex: 'error_msg', key: 'error_msg', width: 220, ellipsis: true },
  ]

  return (
    <PageContainer
      title="定时任务"
      description="定时任务列表，可启停并查看运行记录，数据来自 /v1/cron-tasks"
      extra={<Button onClick={load}>刷新</Button>}
    >
      <Table<CronTask>
        rowKey="id"
        tableLayout="fixed"
        scroll={{ x: 950 }}
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
        width={900}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      >
        <Table<CronTaskRun>
          rowKey="id"
          tableLayout="fixed"
          scroll={{ x: 810 }}
          columns={runColumns}
          dataSource={runs}
          loading={runsLoading}
          pagination={false}
        />
      </Drawer>
    </PageContainer>
  )
}
