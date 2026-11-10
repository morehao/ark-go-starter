import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  App,
  type TableProps,
} from 'antd'
import { PageContainer } from '@ark-go-starter/ui'
import {
  createSetting,
  deleteSetting,
  getSettings,
  updateSetting,
} from '@ark-go-starter/api'
import type { Setting, SettingCreateReq } from '@ark-go-starter/types'

const VALUE_TYPES = ['json', 'toml', 'yaml', 'string', 'int', 'bool', 'float']

/** 状态 Tag 语义色：enabled→success，其余→default */
function StatusTag({ status }: { status: string }) {
  return status === 'enabled' ? (
    <Tag color="success">启用</Tag>
  ) : (
    <Tag color="default">禁用</Tag>
  )
}

export default function SettingsPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<Setting[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Setting | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm<SettingCreateReq>()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getSettings({ page, pageSize })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '加载配置项失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, message])

  useEffect(() => {
    load()
  }, [load])

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
    try {
      const values = await form.validateFields()
      setSubmitting(true)
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
      if (e instanceof Error) message.error(e.message)
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteSetting(id)
      message.success('删除成功')
      load()
    } catch (e) {
      message.error(e instanceof Error ? e.message : '删除失败')
    }
  }

  const columns: TableProps<Setting>['columns'] = [
    { title: '分组', dataIndex: 'group', key: 'group', width: 150, ellipsis: true },
    { title: '键名', dataIndex: 'key', key: 'key', width: 180, ellipsis: true },
    { title: '值类型', dataIndex: 'value_type', key: 'value_type', width: 110 },
    { title: '值', dataIndex: 'value', key: 'value', width: 200, ellipsis: true },
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
      width: 160,
      render: (_, record) => (
        <Space size="middle">
          <Button type="link" size="small" onClick={() => openEdit(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该配置项？"
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

  return (
    <PageContainer
      title="配置中心"
      description="配置项分页列表，支持新建、编辑与删除，数据来自 /v1/demo/settings"
      extra={
        <Space>
          <Button onClick={load}>刷新</Button>
          <Button type="primary" onClick={openCreate}>
            新建配置
          </Button>
        </Space>
      }
    >
      <Table<Setting>
        rowKey="id"
        tableLayout="fixed"
        scroll={{ x: 900 }}
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
          <Form.Item name="description" label="描述">
            <Input placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
