import { useState } from 'react'
import { App, Button, Descriptions, Popconfirm, Space, Tag, Upload, type UploadProps } from 'antd'
import { PageContainer, tokens } from '@ark-go-starter/ui'
import { checkFileExists, deleteFile, getFile, uploadFile } from '@ark-go-starter/api'
import type { FileRecord } from '@ark-go-starter/types'
import {
  MULTIPART_THRESHOLD_BYTES,
  MultipartContentExistsError,
  uploadMultipartFile,
  type MultipartProgress,
} from './multipart'

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

/**
 * 计算小文件的 SHA-256（小写 hex），仅供普通上传路径使用。
 *
 * 后端 POST /v1/demo/files 的 content_hash 是必填项（ginupload.UploadRequest 上
 * binding:"required"）；且当传入值恰好是 64 位 hex 时，filestore 会校验它与实际内容一致
 * （不一致报 content_hash does not match uploaded content）。因此这里必须传真实摘要：
 * 传空 → content_hash is required；传假值 → 失去去重语义、甚至被拒。
 *
 * 限制：crypto.subtle.digest 需要一次性读入整个文件，内存占用与文件同量级；
 * 因此超过 {@link MULTIPART_THRESHOLD_BYTES} 的文件走分片上传（见 ./multipart.ts，
 * 摘要用 js-sha256 增量计算，堆占用只与分块大小相关）。
 */
async function sha256Hex(file: File): Promise<string> {
  if (!globalThis.crypto?.subtle) {
    throw new Error('当前环境不支持 Web Crypto，无法计算文件 sha256（需 https 或 localhost 访问）')
  }
  const digest = await crypto.subtle.digest('SHA-256', await file.arrayBuffer())
  return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, '0')).join('')
}

export default function FilesPage() {
  const { message } = App.useApp()
  const [record, setRecord] = useState<FileRecord | null>(null)
  const [uploading, setUploading] = useState(false)
  const [progressText, setProgressText] = useState<string | null>(null)

  /** 分片进度 → 中文文案（摘要与分片两个阶段分别描述） */
  const handleMultipartProgress = (progress: MultipartProgress) => {
    if (progress.phase === 'hashing') {
      setProgressText(`正在计算文件摘要… ${progress.percent}%`)
      return
    }
    setProgressText(
      `分片上传中 ${progress.partsDone}/${progress.partsTotal}（${progress.percent}%）`,
    )
  }

  // customRequest：接管 antd Upload 的上传流程。
  // - 小文件（≤ 8 MiB）：保持原有普通上传（服务端流式落盘，WebCrypto 一次性算摘要）；
  // - 大文件（> 8 MiB）：走 multipart 分片上传，摘要增量计算、分片逐片直传，
  //   避免 crypto.subtle 把整个文件读进内存（见 ./multipart.ts）。
  const customRequest: UploadProps['customRequest'] = async (options) => {
    const file = options.file as File
    setUploading(true)
    setProgressText(null)
    try {
      let detail: FileRecord
      if (file.size > MULTIPART_THRESHOLD_BYTES) {
        const result = await uploadMultipartFile(file, handleMultipartProgress)
        detail = result.file
        message.success(`上传成功（分片上传 ${result.partsTotal} 片）`)
      } else {
        const uploaded = await uploadFile(file, await sha256Hex(file))
        // 上传成功返回 file_id，再拉一次详情（详情接口返回 name/mime_type/status）
        detail = await getFile(uploaded.file_id).catch(() => uploaded)
        message.success('上传成功')
      }
      setRecord(detail)
      options.onSuccess?.(detail, undefined as never)
    } catch (e) {
      if (e instanceof MultipartContentExistsError) {
        // 同内容已存在（创建会话返回 409）：无需重新上传，直接复用已有记录（秒传）
        const existing = await checkFileExists(e.contentHash).catch(() => null)
        if (existing?.exists && existing.file) {
          setRecord(existing.file)
          message.info('内容已存在，已秒传复用已有文件')
          options.onSuccess?.(existing.file, undefined as never)
          return
        }
      }
      const msg = e instanceof Error ? e.message : '上传失败，请检查后端是否可用'
      message.error(msg)
      options.onError?.(e instanceof Error ? e : new Error(msg))
    } finally {
      setUploading(false)
      setProgressText(null)
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
      description="演示 POST /v1/demo/files 普通上传与 /v1/demo/files/multipart 分片上传（> 8 MB 自动分片），取回文件信息并下载"
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
            单文件上传；超过 8 MB 自动切换分片上传
          </p>
        </Upload.Dragger>

        {uploading && progressText ? (
          <p style={{ margin: 0, color: tokens.textSecondary, fontSize: 12 }} role="status">
            {progressText}
          </p>
        ) : null}

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
