import { useCallback, useEffect, useState } from 'react'
import { Button, Descriptions, Drawer, Space, Table, Tag, Tooltip, App, type DescriptionsProps, type TableProps } from 'antd'
import { HistoryOutlined, RedoOutlined, ReloadOutlined, StopOutlined } from '@ant-design/icons'
import { actionColumn, tableScrollX } from '@morehao/ark-iam-react'
import { PageContainer, tokens, StatCard, StatRow } from '@ark-go-starter/ui'
import { cancelAsyncTask, getAsyncStats, getAsyncTaskRuns, getAsyncTasks, replayAsyncTask } from '@ark-go-starter/api'
import type { AsyncStatsResult, AsyncTask, AsyncTaskRun } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'

/** unix 秒转本地时间字符串，空值显示 '-' */
function fmtTime(v?: number | null): string {
  if (v === undefined || v === null) return '-'
  return new Date(v * 1000).toLocaleString()
}

/** 任务状态 → Tag 语义色与中文文案 */
const STATUS_META: Record<string, { color: string; text: string }> = {
  pending: { color: 'default', text: '待执行' },
  running: { color: 'processing', text: '执行中' },
  succeeded: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  timeout: { color: 'error', text: '超时' },
  canceled: { color: 'warning', text: '已取消' },
  skipped: { color: 'default', text: '已跳过' },
}

/** 状态 Tag */
function StatusTag({ status }: { status: string }) {
  const meta = STATUS_META[status] ?? { color: 'warning', text: status || '-' }
  return <Tag color={meta.color}>{meta.text}</Tag>
}

/** 可取消：pending / running（与后端 CanCancel 一致） */
function canCancel(status: string): boolean {
  return status === 'pending' || status === 'running'
}

/** 可重放：终态（succeeded / failed / timeout / canceled / skipped） */
function canReplay(status: string): boolean {
  return ['succeeded', 'failed', 'timeout', 'canceled', 'skipped'].includes(status)
}

export default function AsyncPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<AsyncTask[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)
  const [stats, setStats] = useState<AsyncStatsResult | null>(null)

  // 尝试记录抽屉
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTitle, setDrawerTitle] = useState('')
  const [currentTask, setCurrentTask] = useState<AsyncTask | null>(null)
  const [runs, setRuns] = useState<AsyncTaskRun[]>([])
  const [runsLoading, setRunsLoading] = useState(false)

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

  const loadStats = useCallback(async () => {
    try {
      setStats(await getAsyncStats())
    } catch {
      // 统计为辅助信息，失败不打断主流程
      setStats(null)
    }
  }, [])

  useEffect(() => {
    void load()
    void loadStats()
  }, [load, loadStats])

  /** 拉取某任务的尝试记录（取消/重放后复用它刷新抽屉） */
  const loadRuns = useCallback(
    async (taskID: string) => {
      setRunsLoading(true)
      try {
        const res = await getAsyncTaskRuns(taskID)
        setRuns(res.list ?? [])
      } catch (e) {
        message.error(localizeError(e, '加载尝试记录失败'))
      } finally {
        setRunsLoading(false)
      }
    },
    [message],
  )

  const openRuns = async (record: AsyncTask) => {
    setDrawerTitle(`尝试记录 · ${record.type || record.id}`)
    setCurrentTask(record)
    setDrawerOpen(true)
    await loadRuns(record.id)
  }

  /** 取消任务（pending/running → canceled） */
  const handleCancel = async (record: AsyncTask) => {
    try {
      const res = await cancelAsyncTask(record.id)
      if (res.canceled) {
        message.success('任务已取消')
      } else {
        message.info('任务已处于终态，无需取消')
      }
      await Promise.all([load(), loadStats()])
      if (drawerOpen && currentTask?.id === record.id) await loadRuns(record.id)
    } catch (e) {
      message.error(localizeError(e, '取消任务失败'))
    }
  }

  /** 重放任务（终态 → pending，generation+1） */
  const handleReplay = async (record: AsyncTask) => {
    try {
      await replayAsyncTask(record.id)
      message.success('任务已重放')
      await Promise.all([load(), loadStats()])
      if (drawerOpen && currentTask?.id === record.id) await loadRuns(record.id)
    } catch (e) {
      message.error(localizeError(e, '重放任务失败'))
    }
  }

  const columns: TableProps<AsyncTask>['columns'] = [
    { title: '任务类型', dataIndex: 'type', key: 'type', width: 180, ellipsis: true },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <StatusTag status={s} /> },
    { title: '业务类型', dataIndex: 'biz_type', key: 'biz_type', width: 120, ellipsis: true },
    { title: '业务 ID', dataIndex: 'biz_id', key: 'biz_id', width: 150, ellipsis: true },
    {
      title: '尝试',
      key: 'attempt',
      width: 90,
      render: (_, r) => `${r.attempt}/${r.max_attempts}`,
    },
    { title: '轮次', dataIndex: 'generation', key: 'generation', width: 70 },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 190, render: (v: number) => fmtTime(v) },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 190, render: (v: number) => fmtTime(v) },
    actionColumn<AsyncTask>({
      max: 3,
      variant: 'icon',
      actions: (record) => [
        {
          key: 'runs',
          label: '尝试记录',
          icon: <HistoryOutlined />,
          onClick: () => void openRuns(record),
        },
        {
          key: 'cancel',
          label: '取消',
          icon: <StopOutlined />,
          danger: true,
          hidden: !canCancel(record.status),
          confirm: '确认取消该任务？',
          onClick: () => void handleCancel(record),
        },
        {
          key: 'replay',
          label: '重放',
          icon: <RedoOutlined />,
          hidden: !canReplay(record.status),
          onClick: () => void handleReplay(record),
        },
      ],
    }),
  ]

  const runColumns: TableProps<AsyncTaskRun>['columns'] = [
    { title: '轮次', dataIndex: 'generation', key: 'generation', width: 70 },
    { title: '尝试', dataIndex: 'attempt', key: 'attempt', width: 70 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <StatusTag status={s} /> },
    { title: '失败分类', dataIndex: 'failure', key: 'failure', width: 110, render: (v: string) => v || '-' },
    { title: '开始时间', dataIndex: 'started_at', key: 'started_at', width: 190, render: (v: number | null | undefined) => fmtTime(v) },
    { title: '结束时间', dataIndex: 'finished_at', key: 'finished_at', width: 190, render: (v: number | null | undefined) => fmtTime(v) },
    { title: '耗时(ms)', dataIndex: 'duration_ms', key: 'duration_ms', width: 100 },
    { title: '错误信息', dataIndex: 'error', key: 'error', width: 220, ellipsis: true },
  ]

  const detailItems: DescriptionsProps['items'] = currentTask
    ? [
        { key: 'id', label: '任务 ID', children: currentTask.id },
        { key: 'type', label: '任务类型', children: currentTask.type || '-' },
        { key: 'status', label: '状态', children: <StatusTag status={currentTask.status} /> },
        { key: 'attempt', label: '尝试', children: `${currentTask.attempt}/${currentTask.max_attempts}` },
        { key: 'gen', label: '轮次', children: currentTask.generation },
        { key: 'next', label: '最早可执行', children: fmtTime(currentTask.next_run_at) },
        { key: 'err', label: '最后错误', children: currentTask.last_error || '-' },
      ]
    : []

  return (
    <PageContainer
      title="异步任务"
      description="任务中心：查看任务状态与尝试记录，支持取消、重放任务，数据来自 /v1/demo/asyncTasks"
      extra={
        <Tooltip title="刷新">
          <Button
            icon={<ReloadOutlined />}
            aria-label="刷新"
            onClick={() => {
              void load()
              void loadStats()
            }}
          />
        </Tooltip>
      }
      summary={
        <StatRow>
          <StatCard label="任务总数" value={(stats?.total ?? total).toLocaleString()} hint="任务中心累计任务数" />
          <StatCard label="待执行" value={(stats?.pending ?? 0).toLocaleString()} hint="pending" />
          <StatCard label="执行中" value={(stats?.running ?? 0).toLocaleString()} hint="running" accent={tokens.success} />
          <StatCard label="失败" value={(stats?.failed ?? 0).toLocaleString()} hint="failed" />
        </StatRow>
      }
    >
      <Table<AsyncTask>
        rowKey="id"
        tableLayout="fixed"
        scroll={tableScrollX(columns)}
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

      <Drawer title={drawerTitle} size={900} open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        <Descriptions
          bordered
          size="small"
          column={2}
          items={detailItems}
          style={{ marginBottom: 16 }}
        />
        <Space size="small" style={{ marginBottom: 16 }}>
          <Button
            danger
            icon={<StopOutlined />}
            disabled={!currentTask || !canCancel(currentTask.status)}
            onClick={() => currentTask && void handleCancel(currentTask)}
          >
            取消任务
          </Button>
          <Button
            icon={<RedoOutlined />}
            disabled={!currentTask || !canReplay(currentTask.status)}
            onClick={() => currentTask && void handleReplay(currentTask)}
          >
            重放任务
          </Button>
        </Space>
        <Table<AsyncTaskRun>
          rowKey="id"
          tableLayout="fixed"
          scroll={{ x: 1000 }}
          columns={runColumns}
          dataSource={runs}
          loading={runsLoading}
          pagination={false}
        />
      </Drawer>
    </PageContainer>
  )
}
