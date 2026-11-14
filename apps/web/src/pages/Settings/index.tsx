import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tooltip,
  App,
  type TableProps,
} from 'antd'
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { StatusSwitch, actionColumn, tableScrollX } from '@morehao/ark-iam-react'
import { PageContainer, tokens, StatCard, StatRow } from '@ark-go-starter/ui'
import {
  createSetting,
  deleteSetting,
  getSettings,
  updateSetting,
  updateSettingStatus,
} from '@ark-go-starter/api'
import type { Setting, SettingCreateReq } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'

const VALUE_TYPES = ['json', 'toml', 'yaml', 'string', 'int', 'bool', 'float']

/** 分组筛选输入框宽度（与 DESIGN.md 搜索区一致） */
const SEARCH_WIDTH = 240

/** 状态筛选项：空值表示不过滤 */
const STATUS_OPTIONS = [
  { label: '启用', value: 'enabled' },
  { label: '禁用', value: 'disabled' },
]

/** 加密模式文案：encrypted→加密，plain→明文；未知值原样展示 */
function encryptionLabel(mode?: string): string {
  if (mode === 'encrypted') return '加密（encrypted）'
  if (mode === 'plain') return '明文（plain）'
  return mode || '-'
}

/** 新建/编辑弹窗表单值：encrypted 仅新建时可填 */
type SettingFormValues = SettingCreateReq

export default function SettingsPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<Setting[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  // 筛选：分组精确匹配（后端 SettingCond.Group 走 Eq），状态精确匹配
  const [groupInput, setGroupInput] = useState('')
  const [groupFilter, setGroupFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Setting | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm<SettingFormValues>()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getSettings({
        page,
        pageSize,
        ...(groupFilter ? { group: groupFilter } : {}),
        ...(statusFilter ? { status: statusFilter } : {}),
      })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(localizeError(e, '加载配置项失败'))
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, groupFilter, statusFilter, message])

  useEffect(() => {
    load()
  }, [load])

  /** 应用分组筛选（Input.Search 回车/点搜索图标触发），页码回到第 1 页 */
  const applyGroupFilter = (value: string) => {
    setPage(1)
    setGroupFilter(value.trim())
  }

  /** 应用状态筛选，页码回到第 1 页 */
  const applyStatusFilter = (value?: string) => {
    setPage(1)
    setStatusFilter(value)
  }

  /** 清空全部筛选条件 */
  const resetFilters = () => {
    setGroupInput('')
    setPage(1)
    setGroupFilter('')
    setStatusFilter(undefined)
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (record: Setting) => {
    setEditing(record)
    form.setFieldsValue({
      group: record.group,
      key: record.key,
      value_type: record.value_type,
      value: record.value,
      description: record.description,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    let values: SettingFormValues
    try {
      values = await form.validateFields()
    } catch {
      // 表单校验失败由 Form.Item 自身提示，无需额外 message
      return
    }
    setSubmitting(true)
    try {
      if (editing) {
        await updateSetting(editing.id, {
          value: values.value,
          description: values.description,
        })
        message.success('更新成功')
      } else {
        await createSetting(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      load()
    } catch (e) {
      message.error(localizeError(e, editing ? '更新配置项失败' : '创建配置项失败'))
    } finally {
      setSubmitting(false)
    }
  }

  /**
   * 状态开关回调：由 `StatusSwitch` 在确认框点「确定」后触发，`nextEnabled` 是目标 on 态。
   * 走 golib ginsetting 的 `PATCH /settings/{id}/status`（只改 status，不触碰值/描述）。
   * 失败时**必须抛出**：StatusSwitch 捕获后不更新受控 checked，开关回退原值并抑制成功提示。
   */
  const handleStatusChange = async (record: Setting, nextEnabled: boolean) => {
    try {
      await updateSettingStatus(record.id, nextEnabled ? 'enabled' : 'disabled')
      // 等列表刷新完成再返回，开关翻转与成功提示的先后顺序才稳定
      await load()
    } catch (e) {
      message.error(localizeError(e, '状态更新失败'))
      throw e
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteSetting(id)
      message.success('删除成功')
      load()
    } catch (e) {
      message.error(localizeError(e, '删除失败'))
    }
  }

  const columns: TableProps<Setting>['columns'] = [
    { title: '分组', dataIndex: 'group', key: 'group', width: 150, ellipsis: true },
    { title: '键名', dataIndex: 'key', key: 'key', width: 180, ellipsis: true },
    { title: '值类型', dataIndex: 'value_type', key: 'value_type', width: 110 },
    { title: '值', dataIndex: 'value', key: 'value', width: 200, ellipsis: true },
    {
      title: '加密模式',
      dataIndex: 'encryption_mode',
      key: 'encryption_mode',
      width: 150,
      render: (v: string) => (
        <span style={{ color: v === 'encrypted' ? tokens.text : tokens.textSecondary }}>
          {encryptionLabel(v)}
        </span>
      ),
    },
    // 可切换状态列走 StatusSwitch（双向确认 + 受控回滚，见 DESIGN §7.2/§8），
    // 走 golib ginsetting 的 PATCH /settings/{id}/status，只改状态不动值/描述。
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
          confirmTitle={record.status === 'enabled' ? '禁用该配置项？' : '启用该配置项？'}
          confirmContent={
            record.status === 'enabled'
              ? '禁用后业务侧读取该配置项时按「未启用」处理，已落库的值不会被删除。'
              : '启用后业务侧读取该配置项时恢复正常取值。'
          }
          okText={record.status === 'enabled' ? '确认禁用' : '确认启用'}
          onConfirmChange={(next) => handleStatusChange(record, next)}
        />
      ),
    },
    // 操作列按 DESIGN §7.3：统一 actionColumn（fixed:'right'），variant:'icon' = 图标 + 文字，
    // 「删除」用 RowAction.confirm 走就地 Popconfirm 二次确认。
    actionColumn<Setting>({
      variant: 'icon',
      actions: (record) => [
        {
          key: 'edit',
          label: '编辑',
          icon: <EditOutlined />,
          onClick: () => openEdit(record),
        },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          icon: <DeleteOutlined />,
          confirm: '确认删除该配置项？',
          onClick: () => void handleDelete(record.id),
        },
      ],
    }),
  ]

  return (
    <PageContainer
      title="配置中心"
      description="配置项分页列表，支持分组/状态筛选、新建（可加密存储）、编辑、启停与删除，数据来自 /v1/demo/settings"
      extra={
        <Space>
          <Tooltip title="刷新">
            <Button icon={<ReloadOutlined />} aria-label="刷新" onClick={load} />
          </Tooltip>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建配置
          </Button>
        </Space>
      }
      summary={
        <StatRow>
          <StatCard label="配置项总数" value={total.toLocaleString()} hint="当前分页接口返回总条数" />
          <StatCard label="本页数量" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} accent={tokens.success} />
        </StatRow>
      }
    >
      <Space wrap size="middle" style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按分组筛选（精确匹配）"
          style={{ width: SEARCH_WIDTH }}
          value={groupInput}
          onChange={(e) => setGroupInput(e.target.value)}
          onSearch={applyGroupFilter}
        />
        <Select
          allowClear
          placeholder="按状态筛选"
          style={{ width: 160 }}
          value={statusFilter}
          options={STATUS_OPTIONS}
          onChange={(v) => applyStatusFilter(v)}
        />
        <Button onClick={resetFilters}>重置</Button>
      </Space>

      <Table<Setting>
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

      <Modal
        title={editing ? '编辑配置项' : '新建配置项'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        forceRender
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="group"
            label="分组"
            rules={[{ required: true, message: '请输入分组' }]}
          >
            <Input placeholder="请输入分组" disabled={!!editing} />
          </Form.Item>
          <Form.Item
            name="key"
            label="键名"
            rules={[{ required: true, message: '请输入键名' }]}
          >
            <Input placeholder="请输入键名" disabled={!!editing} />
          </Form.Item>
          <Form.Item name="value_type" label="值类型" initialValue="string">
            <Select options={VALUE_TYPES.map((t) => ({ label: t, value: t }))} disabled={!!editing} />
          </Form.Item>
          <Form.Item
            name="value"
            label="值"
            rules={[{ required: true, message: '请输入配置值' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入配置值" />
          </Form.Item>
          {!editing ? (
            <Form.Item
              name="encrypted"
              label="加密存储"
              valuePropName="checked"
              initialValue={false}
              extra="开启后配置值加密落库，列表加密模式显示为 encrypted"
            >
              <Switch />
            </Form.Item>
          ) : null}
          <Form.Item name="description" label="描述">
            <Input placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
