import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  App,
  type TableProps,
} from 'antd'
import { PageContainer, StatCard, StatRow, tokens } from '@ark-go-starter/ui'
import {
  createProduct,
  deleteProduct,
  getProducts,
  updateProduct,
} from '@ark-go-starter/api'
import type { ProductCreateReq, ProductPageListItem } from '@ark-go-starter/types'

/** unix 秒转本地时间字符串，空值显示 '-' */
function fmtTime(v?: number | null): string {
  if (v === undefined || v === null) return '-'
  return new Date(v * 1000).toLocaleString()
}

export default function ProductsPage() {
  const { message } = App.useApp()
  const [data, setData] = useState<ProductPageListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  // 筛选条件（提交态）：按名称精确匹配 + 按分类 id 过滤，空值表示不过滤
  const [nameFilter, setNameFilter] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<number | undefined>(undefined)
  const [nameInput, setNameInput] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ProductPageListItem | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm<ProductCreateReq>()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getProducts({
        page,
        pageSize,
        ...(nameFilter ? { name: nameFilter } : {}),
        ...(categoryFilter !== undefined ? { categoryID: categoryFilter } : {}),
      })
      setData(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '加载商品失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, nameFilter, categoryFilter, message])

  useEffect(() => {
    load()
  }, [load])

  /** 应用名称筛选：回车/点搜索时触发，页码回到第 1 页 */
  const applyNameFilter = (v: string) => {
    setNameFilter(v.trim())
    setPage(1)
  }

  /** 应用分类筛选：清空表示不过滤 */
  const applyCategoryFilter = (v: number | undefined) => {
    setCategoryFilter(v)
    setPage(1)
  }

  const resetFilters = () => {
    setNameInput('')
    setNameFilter('')
    setCategoryFilter(undefined)
    setPage(1)
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (record: ProductPageListItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      categoryID: record.categoryID,
      price: record.price,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      if (editing) {
        await updateProduct(editing.productID, values)
        message.success('更新成功')
      } else {
        await createProduct(values)
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

  const handleDelete = async (id: number) => {
    try {
      await deleteProduct(id)
      message.success('删除成功')
      load()
    } catch (e) {
      message.error(e instanceof Error ? e.message : '删除失败')
    }
  }

  const columns: TableProps<ProductPageListItem>['columns'] = [
    { title: 'ID', dataIndex: 'productID', key: 'productID', width: 100 },
    { title: '名称', dataIndex: 'name', key: 'name', width: 200, ellipsis: true },
    {
      title: '分类 ID',
      dataIndex: 'categoryID',
      key: 'categoryID',
      width: 120,
      render: (v: number | undefined) =>
        v === undefined || v === null ? (
          <span style={{ color: tokens.textPlaceholder }}>-</span>
        ) : (
          <Tag color="default">分类 {v}</Tag>
        ),
    },
    {
      title: '价格',
      dataIndex: 'price',
      key: 'price',
      width: 120,
      render: (v: number | undefined) =>
        v === undefined || v === null ? (
          <span style={{ color: tokens.textPlaceholder }}>-</span>
        ) : (
          <span style={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
            ¥{v.toLocaleString()}
          </span>
        ),
    },
    { title: '创建人', dataIndex: 'createdBy', key: 'createdBy', width: 120 },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 200,
      render: (v: number | null | undefined) => fmtTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_, record) => (
        <Space size="middle">
          <Button
            type="link"
            size="small"
            data-testid={`product-edit-${record.productID}`}
            onClick={() => openEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除该商品？"
            onConfirm={() => handleDelete(record.productID)}
          >
            <Button
              type="link"
              size="small"
              danger
              data-testid={`product-delete-${record.productID}`}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="商品管理"
      description="演示后端分页与增删改接口，数据来自 GET /v1/demo/products"
      extra={
        <Space>
          <Button onClick={load}>刷新</Button>
          <Button type="primary" data-testid="product-create-btn" onClick={openCreate}>
            新建商品
          </Button>
        </Space>
      }
      summary={
        <StatRow>
          <StatCard label="商品总数" value={total.toLocaleString()} hint="当前筛选下接口返回总条数" />
          <StatCard
            label="本页数量"
            value={data.length}
            hint={`第 ${page} 页 · 每页 ${pageSize} 条`}
            accent={tokens.success}
          />
          <StatCard
            label="当前筛选"
            value={nameFilter || categoryFilter !== undefined ? '筛选中' : '全部'}
            hint={
              [
                nameFilter ? `名称：${nameFilter}` : '',
                categoryFilter !== undefined ? `分类 ID：${categoryFilter}` : '',
              ]
                .filter(Boolean)
                .join(' · ') || '未设置筛选条件'
            }
            accent={tokens.purple}
          />
        </StatRow>
      }
      toolbar={
        <Space wrap size="middle">
          <Input.Search
            allowClear
            placeholder="按商品名称（精确匹配）"
            style={{ width: 240 }}
            value={nameInput}
            onChange={(e) => setNameInput(e.target.value)}
            onSearch={applyNameFilter}
            data-testid="product-search-name"
          />
          <InputNumber
            placeholder="分类 ID"
            style={{ width: 140 }}
            min={0}
            value={categoryFilter}
            onChange={(v) => applyCategoryFilter(v === null ? undefined : v)}
            data-testid="product-filter-category"
          />
          <Button onClick={resetFilters}>重置</Button>
        </Space>
      }
    >
      <Table<ProductPageListItem>
        rowKey="productID"
        tableLayout="fixed"
        scroll={{ x: 1020 }}
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
        title={editing ? '编辑商品' : '新建商品'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        forceRender
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入商品名称' }]}
          >
            <Input placeholder="请输入商品名称" data-testid="product-input-name" />
          </Form.Item>
          <Form.Item name="categoryID" label="分类 ID">
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              placeholder="请输入分类 ID"
              data-testid="product-input-category"
            />
          </Form.Item>
          <Form.Item name="price" label="价格">
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              placeholder="请输入价格"
              data-testid="product-input-price"
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
