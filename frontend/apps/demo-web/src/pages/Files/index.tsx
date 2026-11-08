import { useState } from 'react'
import { App, Button, Descriptions, Popconfirm, Space, Tag, Upload, type UploadProps } from 'antd'
import { PageContainer, tokens } from '@ark-go-starter/ui'
import { deleteFile, getFile, uploadFile } from '@ark-go-starter/api'
import type { FileRecord } from '@ark-go-starter/types'

/** 文件状态到中文文案 + Tag 颜色的映射（仅状态展示使用语义色） */
const STATUS_MAP: Record<string, { text: string; color: string }> = {
  completed: { text: '已完成', color: tokens.success },
  pending: { text: '待处理', color: tokens.textSecondary },
  uploading: { text: '上传中', color: tokens.warning },
  failed: { text: '失败', color: tokens.error },
  aborted: { text: '已中止', color: tokens.error },
}

/** 文件状态渲染为带色 Tag */
function renderStatus(status: string) {
  const meta = STATUS_MAP[status]
  if (!meta) return <Tag>{status || '-'}</Tag>
  return <Tag color={meta.color}>{meta.text}</Tag>
}

export default function FilesPage() {
  const { message } = App.useApp()
  const [record, setRecord] = useState<FileRecord | null>(null)
  const [uploading, setUploading] = useState(false)

  // customRequest：接管 antd Upload 的上传流程，走 uploadFile，成功后按 file_id 拉详情
  const customRequest: UploadProps['customRequest'] = async (options) => {
    const file = options.file as File
    setUploading(true)
    try {
      const uploaded = await uploadFile(file)
      // 上传成功返回 file_id，再拉一次详情（详情接口返回 name/mime_type/status）
      const detail = await getFile(uploaded.file_id).catch(() => uploaded)
      setRecord(detail)
      message.success('上传成功')
      options.onSuccess?.(detail, undefined as never)
    } catch (e) {
      const msg = e instanceof Error ? e.message : '上传失败，请检查后端是否可用'
      message.error(msg)
      options.onError?.(e instanceof Error ? e : new Error(msg))
    } finally {
      setUploading(false)
    }
  }

  const handleDownload = () => {
    if (!record) return
    // local 存储下最简方式：直接指向 /v1/demo/files/{id}/serve 由浏览器下载
    const link = document.createElement('a')
    link.href = `/v1/demo/files/${record.file_id}/serve`
    link.download = record.name || record.file_id
    link.rel = 'noopener'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleDelete = async () => {
    if (!record) return
    try {
      await deleteFile(record.file_id)
      message.success('删除成功')
      setRecord(null)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '删除失败')
    }
  }

  return (
    <PageContainer
      title="文件上传"
      description="演示通过 POST /v1/demo/files 上传单个文件，取回文件信息并下载"
    >
      <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 640 }}>
        <Upload.Dragger
          name="file"
          multiple={false}
          maxCount={1}
          showUploadList={false}
          customRequest={customRequest}
          disabled={uploading}
        >
          <p style={{ margin: 0, color: tokens.textSecondary, fontSize: 28, lineHeight: 1 }}>
            ⬆
          </p>
          <p style={{ margin: '8px 0 0', color: tokens.text }}>
            点击或拖拽文件到此处上传
          </p>
          <p style={{ margin: '4px 0 0', color: tokens.textPlaceholder, fontSize: 12 }}>
            仅支持单文件普通上传
          </p>
        </Upload.Dragger>

        {record ? (
          <Descriptions
            bordered
            column={1}
            size="small"
            title="文件信息"
            styles={{ label: { width: 120, color: tokens.textSecondary } }}
          >
            <Descriptions.Item label="文件 ID">
              <span style={{ fontFamily: 'ui-monospace, monospace' }}>{record.file_id}</span>
            </Descriptions.Item>
            <Descriptions.Item label="文件名">{record.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="MIME 类型">{record.mime_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">{renderStatus(record.status)}</Descriptions.Item>
            <Descriptions.Item label="操作">
              <Space>
                <Button type="primary" size="small" onClick={handleDownload}>
                  下载
                </Button>
                <Popconfirm title="确认删除该文件？" onConfirm={handleDelete}>
                  <Button size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            </Descriptions.Item>
          </Descriptions>
        ) : null}
      </Space>
    </PageContainer>
  )
}
