import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Drawer,
  InputNumber,
  Popconfirm,
  Space,
  Table,
  Tag,
  App,
  type TableProps,
} from 'antd'
import { PageContainer, tokens, StatCard, StatRow } from '@ark-go-starter/ui'
import {
  cleanupAsyncTaskRuns,
  getAsyncTaskRuns,
  getAsyncTasks,
  markStaleAsyncTaskRuns,
  updateAsyncTaskStatus,
} from '@ark-go-starter/api'
import type { AsyncTask, AsyncTaskRun } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'

/** 陈旧判定默认阈值（分钟）：超过该时长仍 processing 的记录视为卡死 */
const DEFAULT_STALE_MINUTES = 30

/** 清理运行记录默认只删 7 天前的，避免误删近期排障数据 */
const DEFAULT_CLEANUP_DAYS = 7

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

export default function AsyncPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<AsyncTask[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  // 运行记录抽屉（含运维动作：标记陈旧 / 清理）
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTitle, setDrawerTitle] = useState('')
  // 异步任务按 :type 维度操作；列表里的 id 即任务类型名（与现有启停/运行记录调用一致）
  const [currentType, setCurrentType] = useState<string>('')
  const [runs, setRuns] = useState<AsyncTaskRun[]>([])
  const [runsLoading, setRunsLoading] = useState(false)
  const [staleMinutes, setStaleMinutes] = useState(DEFAULT_STALE_MINUTES)
  const [staleSubmitting, setStaleSubmitting] = useState(false)
  const [cleanupDays, setCleanupDays] = useState(DEFAULT_CLEANUP_DAYS)
  const [cleanupSubmitting, setCleanupSubmitting] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getAsyncTasks({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(localizeError(e, '加载异步任务失败'))
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, message])

  useEffect(() => {
    load()
  }, [load])

  const toggleStatus = async (record: AsyncTask) => {
    const next = record.status === 'enabled' ? 'disabled' : 'enabled'
    try {
      await updateAsyncTaskStatus(record.id, next)
      message.success('状态已更新')
      load()
    } catch (e) {
      message.error(localizeError(e, '状态更新失败'))
    }
  }

  /** 拉取某任务类型的运行记录（标记陈旧 / 清理后复用它刷新抽屉） */
  const loadRuns = useCallback(
    async (type: string) => {
      setRunsLoading(true)
      try {
        const res = await getAsyncTaskRuns(type)
        setRuns(res.list ?? [])
      } catch (e) {
        message.error(localizeError(e, '加载运行记录失败'))
      } finally {
        setRunsLoading(false)
      }
    },
    [message],
  )

  const openRuns = async (record: AsyncTask) => {
    setDrawerTitle(`运行记录 · ${record.name || record.id}`)
    setCurrentType(record.id)
    setDrawerOpen(true)
    await loadRuns(record.id)
  }

  /** 标记陈旧 processing 运行记录为 failed（阈值按分钟传入，后端契约是 cutoff_seconds） */
  const handleMarkStale = async () => {
    if (!currentType) return
    setStaleSubmitting(true)
    try {
      const res = await markStaleAsyncTaskRuns(currentType, staleMinutes * 60)
      message.success(`已标记 ${res.affected} 条陈旧运行记录`)
      await loadRuns(currentType)
    } catch (e) {
      message.error(localizeError(e, '标记陈旧运行记录失败'))
    } finally {
      setStaleSubmitting(false)
    }
  }

  /** 清理 N 天前的运行记录（后端契约是 JSON body { before_unix }，不是 query 参数） */
  const handleCleanupRuns = async () => {
    if (!currentType) return
    const beforeUnix = Math.floor(Date.now() / 1000) - cleanupDays * 86400
    setCleanupSubmitting(true)
    try {
      const res = await cleanupAsyncTaskRuns(currentType, beforeUnix)
      message.success(`已清理 ${res.affected} 条运行记录`)
      await loadRuns(currentType)
    } catch (e) {
      message.error(localizeError(e, '清理运行记录失败'))
    } finally {
      setCleanupSubmitting(false)
    }
  }

  const columns: TableProps<AsyncTask>['columns'] = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 180, ellipsis: true },
    { title: '描述', dataIndex: 'description', key: 'description', width: 260, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <StatusTag status={s} />,
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
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

  const runColumns: TableProps<AsyncTaskRun>['columns'] = [
    { title: '队列', dataIndex: 'queue', key: 'queue', width: 140, ellipsis: true },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <StatusTag status={s} /> },
    {
      title: '重试',
      key: 'retried',
      width: 100,
      render: (_, r) => `${r.retried}/${r.max_retry}`,
    },
    { title: '开始时间', dataIndex: 'start_at', key: 'start_at', width: 190, render: (v: number | null | undefined) => fmtTime(v) },
    { title: '耗时(ms)', dataIndex: 'duration_ms', key: 'duration_ms', width: 110 },
    { title: '错误信息', dataIndex: 'error_msg', key: 'error_msg', width: 220, ellipsis: true },
  ]

  return (
    <PageContainer
      title="异步任务"
      description="异步任务列表，支持启停、查看运行记录，以及标记陈旧运行与清理历史运行记录，数据来自 /v1/demo/async-tasks"
      extra={<Button onClick={load}>刷新</Button>}
      summary={
        <StatRow>
          <StatCard label="任务总数" value={total.toLocaleString()} hint="当前分页接口返回总条数" />
          <StatCard label="本页数量" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} accent={tokens.success} />
        </StatRow>
      }
    >
      <Table<AsyncTask>
        rowKey="id"
        tableLayout="fixed"
        scroll={{ x: 760 }}
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
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 16,
            marginBottom: 16,
          }}
        >
          <Space size="small">
            <span style={{ color: tokens.textSecondary }}>陈旧判定（分钟）</span>
            <InputNumber
              min={1}
              max={43200}
              style={{ width: 120 }}
              value={staleMinutes}
              onChange={(v) => setStaleMinutes(typeof v === 'number' ? v : staleMinutes)}
            />
            <Button loading={staleSubmitting} onClick={handleMarkStale}>
              标记陈旧
            </Button>
          </Space>
          <Space size="small">
            <span style={{ color: tokens.textSecondary }}>清理早于（天前）</span>
            <InputNumber
              min={1}
              max={3650}
              style={{ width: 120 }}
              value={cleanupDays}
              onChange={(v) => setCleanupDays(typeof v === 'number' ? v : cleanupDays)}
            />
            <Popconfirm
              title="确认清理该时间之前的运行记录？"
              onConfirm={handleCleanupRuns}
            >
              <Button danger loading={cleanupSubmitting}>
                清理运行记录
              </Button>
            </Popconfirm>
          </Space>
        </div>
        <Table<AsyncTaskRun>
          rowKey="id"
          tableLayout="fixed"
          scroll={{ x: 860 }}
          columns={runColumns}
          dataSource={runs}
          loading={runsLoading}
          pagination={false}
        />
      </Drawer>
    </PageContainer>
  )
}
