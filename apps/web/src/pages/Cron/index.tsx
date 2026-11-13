import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Descriptions,
  Drawer,
  InputNumber,
  Popconfirm,
  Space,
  Table,
  Tag,
  App,
  type DescriptionsProps,
  type TableProps,
} from 'antd'
import { PageContainer, tokens, StatCard, StatRow } from '@ark-go-starter/ui'
import {
  cleanupCronTaskRuns,
  getCronTask,
  getCronTaskRuns,
  getCronTasks,
  markStaleCronTaskRuns,
  updateCronTaskStatus,
} from '@ark-go-starter/api'
import type { CronTask, CronTaskRun } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'

/** 陈旧判定默认阈值（分钟）：超过该时长仍 running 的记录视为卡死 */
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

export default function CronPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<CronTask[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  // 任务详情抽屉
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailTitle, setDetailTitle] = useState('')
  const [detail, setDetail] = useState<CronTask | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  // 运行记录抽屉（含运维动作：标记陈旧 / 清理）
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTitle, setDrawerTitle] = useState('')
  const [currentTask, setCurrentTask] = useState<CronTask | null>(null)
  const [runs, setRuns] = useState<CronTaskRun[]>([])
  const [runsLoading, setRunsLoading] = useState(false)
  const [staleMinutes, setStaleMinutes] = useState(DEFAULT_STALE_MINUTES)
  const [staleSubmitting, setStaleSubmitting] = useState(false)
  const [cleanupDays, setCleanupDays] = useState(DEFAULT_CLEANUP_DAYS)
  const [cleanupSubmitting, setCleanupSubmitting] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getCronTasks({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(localizeError(e, '加载定时任务失败'))
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
      message.error(localizeError(e, '状态更新失败'))
    }
  }

  /** 拉取某任务的运行记录（标记陈旧 / 清理后复用它刷新抽屉） */
  const loadRuns = useCallback(
    async (task: CronTask) => {
      setRunsLoading(true)
      try {
        const res = await getCronTaskRuns(task.id)
        setRuns(res.list ?? [])
      } catch (e) {
        message.error(localizeError(e, '加载运行记录失败'))
      } finally {
        setRunsLoading(false)
      }
    },
    [message],
  )

  const openRuns = async (record: CronTask) => {
    setDrawerTitle(`运行记录 · ${record.name || record.id}`)
    setCurrentTask(record)
    setDrawerOpen(true)
    await loadRuns(record)
  }

  const openDetail = async (record: CronTask) => {
    setDetailTitle(`任务详情 · ${record.name || record.id}`)
    setDetail(null)
    setDetailOpen(true)
    setDetailLoading(true)
    try {
      const res = await getCronTask(record.id)
      setDetail(res)
    } catch (e) {
      message.error(localizeError(e, '加载任务详情失败'))
    } finally {
      setDetailLoading(false)
    }
  }

  /** 标记陈旧 running 运行记录为 failed（阈值按分钟传入，后端契约是 cutoff_seconds） */
  const handleMarkStale = async () => {
    if (!currentTask) return
    setStaleSubmitting(true)
    try {
      const res = await markStaleCronTaskRuns(currentTask.id, staleMinutes * 60)
      message.success(`已标记 ${res.affected} 条陈旧运行记录`)
      await loadRuns(currentTask)
    } catch (e) {
      message.error(localizeError(e, '标记陈旧运行记录失败'))
    } finally {
      setStaleSubmitting(false)
    }
  }

  /** 清理 N 天前的运行记录（后端契约是 JSON body { before_unix }，不是 query 参数） */
  const handleCleanupRuns = async () => {
    if (!currentTask) return
    const beforeUnix = Math.floor(Date.now() / 1000) - cleanupDays * 86400
    setCleanupSubmitting(true)
    try {
      const res = await cleanupCronTaskRuns(currentTask.id, beforeUnix)
      message.success(`已清理 ${res.affected} 条运行记录`)
      await loadRuns(currentTask)
    } catch (e) {
      message.error(localizeError(e, '清理运行记录失败'))
    } finally {
      setCleanupSubmitting(false)
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
      width: 220,
      render: (_, record) => (
        <span style={{ display: 'flex', gap: 16 }}>
          <Button type="link" size="small" onClick={() => openDetail(record)}>
            详情
          </Button>
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

  const detailItems: DescriptionsProps['items'] = detail
    ? [
        { key: 'id', label: '任务 ID', children: detail.id },
        { key: 'name', label: '名称', children: detail.name || '-' },
        { key: 'task_type', label: '任务类型', children: detail.task_type || '-' },
        { key: 'biz_type', label: '业务类型', children: detail.biz_type || '-' },
        { key: 'biz_id', label: '业务 ID', children: detail.biz_id || '-' },
        { key: 'spec', label: 'Cron 表达式', children: detail.spec || '-' },
        { key: 'status', label: '状态', children: <StatusTag status={detail.status} /> },
        { key: 'last_run_at', label: '上次运行时间', children: fmtTime(detail.last_run_at) },
        { key: 'next_run_at', label: '下次运行时间', children: fmtTime(detail.next_run_at) },
        { key: 'description', label: '说明', children: detail.description || '-' },
        { key: 'created_at', label: '创建时间', children: fmtTime(detail.created_at) },
        { key: 'updated_at', label: '更新时间', children: fmtTime(detail.updated_at) },
      ]
    : []

  return (
    <PageContainer
      title="定时任务"
      description="定时任务列表，支持查看详情、运行记录、启停，以及标记陈旧运行与清理历史运行记录，数据来自 /v1/demo/cron-tasks"
      extra={<Button onClick={load}>刷新</Button>}
      summary={
        <StatRow>
          <StatCard label="任务总数" value={total.toLocaleString()} hint="当前分页接口返回总条数" />
          <StatCard label="本页数量" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} accent={tokens.success} />
        </StatRow>
      }
    >
      <Table<CronTask>
        rowKey="id"
        tableLayout="fixed"
        scroll={{ x: 990 }}
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
        title={detailTitle}
        size={720}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        loading={detailLoading}
      >
        {detail ? <Descriptions bordered size="small" column={1} items={detailItems} /> : null}
      </Drawer>

      <Drawer
        title={drawerTitle}
        size={900}
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
