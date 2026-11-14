import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
  App,
  type DescriptionsProps,
  type TableProps,
} from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons'
import { StatusSwitch, actionColumn, tableScrollX } from '@morehao/ark-iam-react'
import { PageContainer, tokens, StatCard, StatRow } from '@ark-go-starter/ui'
import {
  checkDictTypeIntegrity,
  createDictType,
  deleteDictType,
  getDictItems,
  getDictTypes,
  updateDictType,
  updateDictTypeStatus,
} from '@ark-go-starter/api'
import type {
  DictIntegrityIssue,
  DictIntegrityReport,
  DictItem,
  DictType,
} from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'

/** 新建/编辑弹出层的表单值（编辑态 code/status 不可改，后端 update 也不接收） */
interface DictTypeFormValues {
  // 类型编码（仅新建可填）
  code: string
  // 类型名称
  name?: string
  // 状态：enabled / disabled（仅新建可填）
  status?: string
  // 说明
  description?: string
  // 类型级扩展 JSON 字符串
  extra?: string
}

/** 巡检问题类型（golib dict 常量）→ 中文标签 */
const ISSUE_KIND_LABELS: Record<string, string> = {
  path_mismatch: '路径不一致',
  level_mismatch: '层级不一致',
  orphan: '孤儿节点',
  cycle: '父子成环',
  level_overflow: '层级超限',
  type_mismatch: '类型失联',
}

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

  // 字典项抽屉
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTitle, setDrawerTitle] = useState('')
  const [items, setItems] = useState<DictItem[]>([])
  const [itemsLoading, setItemsLoading] = useState(false)

  // 新建/编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<DictType | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm<DictTypeFormValues>()

  // 完整性检查抽屉
  const [integrityOpen, setIntegrityOpen] = useState(false)
  const [integrityTitle, setIntegrityTitle] = useState('')
  const [integrity, setIntegrity] = useState<DictIntegrityReport | null>(null)
  const [integrityLoading, setIntegrityLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getDictTypes({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(localizeError(e, '加载字典类型失败'))
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, message])

  useEffect(() => {
    load()
  }, [load])

  /**
   * 状态开关回调：由 `StatusSwitch` 在确认框点「确定」后触发，`nextEnabled` 是目标 on 态。
   * 成功后重新拉取列表（状态列文案随之翻转）；失败时**必须抛出**——StatusSwitch 内部捕获后
   * 不更新受控 checked，开关自动回退到原值，同时抑制它自带的成功提示。
   */
  const handleStatusChange = async (record: DictType, nextEnabled: boolean) => {
    try {
      await updateDictTypeStatus(record.id, nextEnabled ? 'enabled' : 'disabled')
      // 等列表刷新完成再返回，开关翻转与成功提示的先后顺序才稳定
      await load()
    } catch (e) {
      message.error(localizeError(e, '状态更新失败'))
      throw e
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteDictType(id)
      message.success('删除成功')
      load()
    } catch (e) {
      message.error(localizeError(e, '删除失败'))
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
      message.error(localizeError(e, '加载字典项失败'))
    } finally {
      setItemsLoading(false)
    }
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (record: DictType) => {
    setEditing(record)
    form.setFieldsValue({
      code: record.code,
      name: record.name,
      status: record.status,
      description: record.description,
      extra: record.extra,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    let values: DictTypeFormValues
    try {
      values = await form.validateFields()
    } catch {
      // 表单校验失败由 Form.Item 自身提示，无需额外 message
      return
    }
    setSubmitting(true)
    try {
      if (editing) {
        await updateDictType(editing.id, {
          name: values.name,
          description: values.description,
          extra: values.extra,
        })
        message.success('更新成功')
      } else {
        await createDictType({
          code: values.code,
          name: values.name,
          status: values.status,
          description: values.description,
          extra: values.extra,
        })
        message.success('创建成功')
      }
      setModalOpen(false)
      load()
    } catch (e) {
      message.error(localizeError(e, editing ? '更新字典类型失败' : '创建字典类型失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const openIntegrity = async (record: DictType) => {
    setIntegrityTitle(`完整性检查 · ${record.name || record.code}`)
    setIntegrity(null)
    setIntegrityOpen(true)
    setIntegrityLoading(true)
    try {
      const report = await checkDictTypeIntegrity(record.id)
      setIntegrity(report)
    } catch (e) {
      message.error(localizeError(e, '完整性检查失败'))
    } finally {
      setIntegrityLoading(false)
    }
  }

  const columns: TableProps<DictType>['columns'] = [
    { title: '编码', dataIndex: 'code', key: 'code', width: 160, ellipsis: true },
    { title: '名称', dataIndex: 'name', key: 'name', width: 160, ellipsis: true },
    // 可切换状态列走 StatusSwitch（双向确认 + 受控回滚，见 DESIGN §7.2/§8），
    // 操作列因此不再放「启用/禁用」按钮。
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (_: string, record) => (
        <StatusSwitch
          checked={record.status === 'enabled'}
          checkedChildren="启用"
          unCheckedChildren="禁用"
          confirmTitle={record.status === 'enabled' ? '禁用该字典类型？' : '启用该字典类型？'}
          confirmContent={
            record.status === 'enabled'
              ? '禁用后该字典类型及其字典项在业务下拉中不再可选，已引用的历史数据不受影响。'
              : '启用后该字典类型及其字典项将重新在业务下拉中可选。'
          }
          okText={record.status === 'enabled' ? '确认禁用' : '确认启用'}
          onConfirmChange={(next) => handleStatusChange(record, next)}
        />
      ),
    },
    { title: '说明', dataIndex: 'description', key: 'description', width: 200, ellipsis: true },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 190,
      render: (v: number) => fmtTime(v),
    },
    // 操作列按 DESIGN §7.3：4 个操作超过 max=3，横排保留 2 个高频项，「完整性检查 / 删除」
    // 收进「更多」下拉（菜单项纵向排列，避免行内按钮挤成一团）；「启用/禁用」已上移为状态列开关。
    actionColumn<DictType>({
      max: 3,
      variant: 'icon',
      actions: (record) => [
        {
          key: 'items',
          label: '查看项',
          icon: <UnorderedListOutlined />,
          onClick: () => void openItems(record),
        },
        {
          key: 'edit',
          label: '编辑',
          icon: <EditOutlined />,
          onClick: () => openEdit(record),
        },
        {
          key: 'integrity',
          label: '完整性检查',
          icon: <SafetyCertificateOutlined />,
          onClick: () => void openIntegrity(record),
        },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          icon: <DeleteOutlined />,
          confirm: '确认删除该字典类型？',
          onClick: () => void handleDelete(record.id),
        },
      ],
    }),
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

  const issueColumns: TableProps<DictIntegrityIssue>['columns'] = [
    {
      title: '项 ID',
      dataIndex: 'item_id',
      key: 'item_id',
      width: 340,
      ellipsis: true,
      render: (v: string) => <Typography.Text>{v}</Typography.Text>,
    },
    { title: '码值', dataIndex: 'value', key: 'value', width: 140, ellipsis: true },
    {
      title: '问题类型',
      dataIndex: 'kind',
      key: 'kind',
      width: 120,
      render: (v: string) => ISSUE_KIND_LABELS[v] ?? v,
    },
    { title: '详情', dataIndex: 'detail', key: 'detail', width: 220, ellipsis: true },
  ]

  const integrityItems: DescriptionsProps['items'] = integrity
    ? [
        { key: 'type_code', label: '类型编码', children: integrity.type_code },
        { key: 'checked', label: '检查项数', children: integrity.checked },
        {
          key: 'healthy',
          label: '健康状态',
          children: integrity.healthy ? (
            <Tag color="success">健康</Tag>
          ) : (
            <Tag color="error">异常</Tag>
          ),
        },
        {
          key: 'issues',
          label: '问题数',
          children: integrity.issues.length,
        },
        { key: 'orphans', label: '孤儿节点数', children: integrity.orphans.length },
      ]
    : []

  return (
    <PageContainer
      title="数据字典"
      description="字典类型分页列表，支持新建/编辑、查看字典项、完整性检查与启停删除，数据来自 /v1/demo/dict-types"
      extra={
        <Space>
          <Tooltip title="刷新">
            <Button icon={<ReloadOutlined />} aria-label="刷新" onClick={load} />
          </Tooltip>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建字典类型
          </Button>
        </Space>
      }
      summary={
        <StatRow>
          <StatCard label="字典类型总数" value={total.toLocaleString()} hint="当前分页接口返回总条数" />
          <StatCard label="本页数量" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} accent={tokens.success} />
        </StatRow>
      }
    >
      <Table<DictType>
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

      <Drawer
        title={drawerTitle}
        size={820}
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

      <Drawer
        title={integrityTitle}
        size={860}
        open={integrityOpen}
        onClose={() => setIntegrityOpen(false)}
      >
        <Spin spinning={integrityLoading}>
          {integrity ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <Descriptions bordered size="small" column={2} items={integrityItems} />
              {integrity.orphans.length > 0 ? (
                <div
                  style={{
                    border: `1px solid ${tokens.warningBorder}`,
                    background: tokens.warningBg,
                    borderRadius: tokens.radiusLg,
                    padding: '10px 12px',
                    color: tokens.text,
                    wordBreak: 'break-all',
                  }}
                >
                  孤儿节点（parent_id 指向不存在的项）：{integrity.orphans.join('、')}
                </div>
              ) : null}
              <Table<DictIntegrityIssue>
                rowKey={(r) => `${r.item_id}-${r.kind}`}
                tableLayout="fixed"
                scroll={{ x: 820 }}
                columns={issueColumns}
                dataSource={integrity.issues}
                pagination={false}
                locale={{ emptyText: '未发现结构性问题' }}
              />
            </div>
          ) : null}
        </Spin>
      </Drawer>

      <Modal
        title={editing ? '编辑字典类型' : '新建字典类型'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        forceRender
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="code"
            label="类型编码"
            rules={[{ required: true, message: '请输入类型编码' }]}
          >
            <Input placeholder="请输入类型编码" disabled={!!editing} />
          </Form.Item>
          <Form.Item name="name" label="类型名称">
            <Input placeholder="请输入类型名称" />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="enabled">
            <Select
              disabled={!!editing}
              options={[
                { label: '启用', value: 'enabled' },
                { label: '禁用', value: 'disabled' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input placeholder="请输入说明" />
          </Form.Item>
          <Form.Item name="extra" label="扩展 JSON">
            <Input.TextArea rows={2} placeholder="可选，合法的 JSON 字符串" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
