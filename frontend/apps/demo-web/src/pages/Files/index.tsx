import { useState } from 'react'
import {
  App,
  Button,
  Descriptions,
  Input,
  Popconfirm,
  Space,
  Tag,
  Tooltip,
  Upload,
  type UploadProps,
} from 'antd'
import { PageContainer, tokens } from '@ark-go-starter/ui'
import {
  checkFileExists,
  deleteFile,
  getFileDetail,
  serveFileURL,
  uploadFile,
} from '@ark-go-starter/api'
import type { FileDetail, FileRecord } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'
import {
  MULTIPART_THRESHOLD_BYTES,
  MultipartContentExistsError,
  uploadMultipartFile,
  type MultipartProgress,
} from './multipart'
import AccessMethods from './AccessMethods'
import PartsDebug from './PartsDebug'

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

/** 字节数格式化为 B/KB/MB（保留两位小数，去掉多余尾零） */
function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '-'
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes / 1024
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  const rounded = Math.round(value * 100) / 100
  return `${rounded} ${units[unitIndex]}`
}

/** 等宽字体样式（ID / URL / hash 专用，见 DESIGN.md §3） */
const monoStyle = { fontFamily: 'ui-monospace, monospace' as const }

/** 详情区字段的通用标签列样式 */
const labelStyle = { width: 120, color: tokens.textSecondary }

export default function FilesPage() {
  const { message } = App.useApp()
  // 当前展示的文件完整详情（上传成功后自动填充；也可通过「按文件 ID 查询」单独拉取）
  const [detail, setDetail] = useState<FileDetail | null>(null)
  // 上传中状态（禁用拖拽区）
  const [uploading, setUploading] = useState(false)
  // 分片上传进度文案
  const [progressText, setProgressText] = useState<string | null>(null)
  // 「按文件 ID 查询」输入框内容
  const [queryId, setQueryId] = useState('')
  // 详情查询中状态
  const [detailLoading, setDetailLoading] = useState(false)

  /** 拉取并填充详情区（供上传成功、刷新、按 ID 查询复用） */
  const loadDetail = async (fileId: string, silent = false) => {
    setDetailLoading(true)
    try {
      const data = await getFileDetail(fileId)
      setDetail(data)
      return data
    } catch (e) {
      if (!silent) message.error(localizeError(e, '文件不存在或已删除'))
      return null
    } finally {
      setDetailLoading(false)
    }
  }

  /** 分片进度 → 中文文案（摘要与分片两个阶段分别描述） */
  const handleMultipartProgress = (progress: MultipartProgress) => {
    if (progress.phase === 'hashing') {
      setProgressText(`正在计算文件摘要… ${progress.percent}%`)
      return
    }
    setProgressText(`分片上传中 ${progress.partsDone}/${progress.partsTotal}（${progress.percent}%）`)
  }

  // customRequest：接管 antd Upload 的上传流程。
  // - 小文件（≤ 8 MiB）：先算摘要 → 上传前 check-exist 探测秒传 → 命中则直接复用；
  //   未命中才走普通上传（服务端流式落盘）；
  // - 大文件（> 8 MiB）：走 multipart 分片上传，摘要增量计算、分片逐片直传，
  //   避免 crypto.subtle 把整个文件读进内存（见 ./multipart.ts，内部已含 409→秒传复用）。
  const customRequest: UploadProps['customRequest'] = async (options) => {
    const file = options.file as File
    setUploading(true)
    setProgressText(null)
    try {
      let uploadedRecord: FileRecord
      if (file.size > MULTIPART_THRESHOLD_BYTES) {
        const result = await uploadMultipartFile(file, handleMultipartProgress)
        uploadedRecord = result.file
        message.success(`上传成功（分片上传 ${result.partsTotal} 片）`)
      } else {
        // 小文件先算摘要（原本上传也需要它），先做 check-exist 秒传探测，命中则不再上传
        const hash = await sha256Hex(file)
        const exist = await checkFileExists(hash).catch(() => null)
        if (exist?.exists && exist.file) {
          uploadedRecord = exist.file
          message.success('上传成功')
          message.info('内容已存在，已秒传复用')
        } else {
          uploadedRecord = await uploadFile(file, hash)
          message.success('上传成功')
        }
      }
      // 上传成功后自动拉取完整详情填充详情区（详情失败时用上传返回的窄记录兜底回调）
      const full = await loadDetail(uploadedRecord.file_id, true)
      options.onSuccess?.(full ?? uploadedRecord, undefined as never)
    } catch (e) {
      if (e instanceof MultipartContentExistsError) {
        // 同内容已存在（创建会话返回 409）：无需重新上传，直接复用已有记录（秒传）
        const existing = await checkFileExists(e.contentHash).catch(() => null)
        if (existing?.exists && existing.file) {
          message.info('内容已存在，已秒传复用已有文件')
          await loadDetail(existing.file.file_id, true)
          options.onSuccess?.(existing.file, undefined as never)
          return
        }
      }
      message.error(localizeError(e, '上传失败，请检查后端是否可用'))
      options.onError?.(e instanceof Error ? e : new Error('上传失败'))
    } finally {
      setUploading(false)
      setProgressText(null)
    }
  }

  // 下载：用隐藏 <a download> 指向 serve 直出端点（e2e 依赖 download.url() 含 /serve）
  const handleDownload = () => {
    if (!detail) return
    const link = document.createElement('a')
    link.href = serveFileURL(detail.file_id)
    link.download = detail.name || detail.file_id
    link.rel = 'noopener'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const handleDelete = async () => {
    if (!detail) return
    try {
      await deleteFile(detail.file_id)
      message.success('删除成功')
      setDetail(null)
    } catch (e) {
      message.error(localizeError(e, '删除失败'))
    }
  }

  /** 刷新详情：重新按当前 id 拉取 */
  const handleRefresh = async () => {
    if (!detail) return
    const fresh = await loadDetail(detail.file_id)
    if (fresh) message.success('详情已刷新')
  }

  /** 按文件 ID 查询详情（可验证详情能力独立调用） */
  const handleQueryDetail = async () => {
    const id = queryId.trim()
    if (!id) {
      message.warning('请输入文件 ID')
      return
    }
    const data = await loadDetail(id)
    if (data) message.success('查询成功')
  }

  return (
    <PageContainer
      title="文件上传"
      description="演示 golib ginupload 的完整路由能力：普通上传 / 分片上传（> 8 MB 自动分片）、文件详情、四种取回方式（serve / redirect / presign-url / query）与分片会话调试"
    >
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* ① 上传区 */}
        <Upload.Dragger
          name="file"
          multiple={false}
          maxCount={1}
          showUploadList={false}
          customRequest={customRequest}
          disabled={uploading}
          style={{ maxWidth: 640 }}
        >
          <p style={{ margin: 0, color: tokens.textSecondary, fontSize: 28, lineHeight: 1 }}>⬆</p>
          <p style={{ margin: '8px 0 0', color: tokens.text }}>点击或拖拽文件到此处上传</p>
          <p style={{ margin: '4px 0 0', color: tokens.textPlaceholder, fontSize: 12 }}>
            单文件上传；超过 8 MB 自动切换分片上传；同内容文件自动秒传复用
          </p>
        </Upload.Dragger>

        {uploading && progressText ? (
          <p style={{ margin: 0, color: tokens.textSecondary, fontSize: 12 }} role="status">
            {progressText}
          </p>
        ) : null}

        {/* ② 文件详情（仅在存在详情时渲染，避免 e2e 在上传失败时误判「文件信息」出现） */}
        {detail ? (
          <Descriptions
            bordered
            column={1}
            size="small"
            title="文件信息"
            styles={{ label: labelStyle }}
          >
            <Descriptions.Item label="文件 ID">
              <span style={monoStyle}>{detail.file_id}</span>
            </Descriptions.Item>
            <Descriptions.Item label="内容哈希">
              <Tooltip title={detail.content_hash}>
                <span style={{ ...monoStyle, wordBreak: 'break-all' }}>
                  {detail.content_hash || '-'}
                </span>
              </Tooltip>
            </Descriptions.Item>
            <Descriptions.Item label="文件名">{detail.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="文件大小">
              {formatSize(detail.size)}
              <span style={{ color: tokens.textPlaceholder, marginLeft: 8 }}>
                （{detail.size} 字节）
              </span>
            </Descriptions.Item>
            <Descriptions.Item label="MIME 类型">{detail.mime_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="存储 URI">
              <Tooltip title={detail.storage_uri}>
                <span
                  style={{
                    ...monoStyle,
                    display: 'inline-block',
                    maxWidth: '100%',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    verticalAlign: 'bottom',
                  }}
                >
                  {detail.storage_uri || '-'}
                </span>
              </Tooltip>
            </Descriptions.Item>
            {detail.upload_id ? (
              <Descriptions.Item label="上传会话 ID">
                <span style={monoStyle}>{detail.upload_id}</span>
              </Descriptions.Item>
            ) : null}
            <Descriptions.Item label="状态">{renderStatus(detail.status)}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{detail.created_at || '-'}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{detail.updated_at || '-'}</Descriptions.Item>
            <Descriptions.Item label="操作">
              <Space wrap>
                <Button size="small" onClick={handleDownload}>
                  下载
                </Button>
                <Button size="small" loading={detailLoading} onClick={handleRefresh}>
                  刷新详情
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

        {/* 按文件 ID 查询详情：验证详情接口可独立调用 */}
        <div style={{ maxWidth: 640 }}>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              placeholder="输入任意文件 ID 查询详情"
              value={queryId}
              allowClear
              onChange={(e) => setQueryId(e.target.value)}
              onPressEnter={handleQueryDetail}
              style={monoStyle}
            />
            <Button type="primary" loading={detailLoading} onClick={handleQueryDetail}>
              查询
            </Button>
          </Space.Compact>
        </div>

        {/* ③ 下载与访问方式（仅在有详情时展示，针对当前文件；key 随 file_id 变化以重置本地状态） */}
        {detail ? (
          <AccessMethods
            key={detail.file_id}
            fileId={detail.file_id}
            storageUri={detail.storage_uri}
          />
        ) : null}

        {/* ④ 分片会话调试（key 随当前详情 id 变化以重置输入与列举结果） */}
        <PartsDebug key={detail?.file_id ?? '__empty__'} defaultFileId={detail?.file_id ?? ''} />
      </Space>
    </PageContainer>
  )
}
