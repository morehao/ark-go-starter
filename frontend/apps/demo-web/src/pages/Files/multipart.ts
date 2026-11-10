import { sha256 } from 'js-sha256'
import {
  abortMultipartUpload,
  completeMultipartUpload,
  createMultipartUpload,
  presignUploadPart,
  type MultipartPart,
} from '@ark-go-starter/api'
import type { FileRecord, PresignURLResp } from '@ark-go-starter/types'

/**
 * 分片上传（multipart）客户端实现，供 Files 页在超过阈值时使用。
 *
 * 为什么需要它：普通上传（POST /demo/files）虽然由服务端流式落盘，但前端必须先算
 * 整个文件的 SHA-256 作为 content_hash（后端 binding:"required"，且 64 位 hex 会与
 * 内容比对）。crypto.subtle.digest 只能一次性吃完整 buffer，超大文件会占用等量内存。
 * 分片上传的 content_hash 同样是**整个文件**的摘要，所以这里配合 js-sha256 的增量
 * 接口分块计算，堆占用只与分块大小相关；上传阶段则逐片取预签名地址直传。
 *
 * 契约（golib ginupload /files/multipart）：
 *   POST /v1/demo/files/multipart                 → {file_id, upload_id}（content_hash 必填）
 *   POST /v1/demo/files/multipart/:id/parts       → {url, method, headers, expires_in}
 *   PUT  <预签名 url>                              → {part_number, etag}
 *   POST /v1/demo/files/multipart/:id/complete    → FileRecord
 *   DELETE /v1/demo/files/multipart/:id           → 取消并清理分片
 */

/** 超过该体积的文件走分片上传（8 MiB）；小文件保持原有的普通上传路径。 */
export const MULTIPART_THRESHOLD_BYTES = 8 * 1024 * 1024

/** 分片大小（8 MiB）：与阈值一致，略超阈值的文件正好两片。 */
const PART_SIZE_BYTES = 8 * 1024 * 1024

/** 增量计算摘要时每次读入的分块（2 MiB），堆占用与文件体积无关。 */
const HASH_CHUNK_BYTES = 2 * 1024 * 1024

/** 摘要计算阶段占总进度的权重（%），剩余进度给分片上传。 */
const HASH_PROGRESS_WEIGHT = 10

/** 分片上传进度：phase 区分「算摘要」与「传分片」两个阶段。 */
export interface MultipartProgress {
  // 当前阶段
  phase: 'hashing' | 'uploading'
  // 总体进度（0-100）
  percent: number
  // 已完成分片数
  partsDone: number
  // 分片总数
  partsTotal: number
}

/** 分片上传成功结果 */
export interface MultipartUploadResult {
  // 服务端返回的文件记录（file_id/name/mime_type/status）
  file: FileRecord
  // 本次计算出的整个文件 SHA-256
  contentHash: string
  // 分片总数
  partsTotal: number
}

/**
 * 创建会话时后端以 HTTP 409 + "content already exists" 表示同内容已存在
 * （filestore.ErrContentExists）。页面捕获该错误后走 check-exist 秒传复用，
 * 而不是把 409 当作失败展示。
 */
export class MultipartContentExistsError extends Error {
  readonly contentHash: string

  constructor(contentHash: string) {
    super('内容已存在')
    this.name = 'MultipartContentExistsError'
    this.contentHash = contentHash
  }
}

/** 判断 axios 抛出的错误是否是指定 HTTP 状态码（不引入 axios 依赖的窄类型守卫）。 */
function isHTTPStatus(error: unknown, status: number): boolean {
  if (typeof error !== 'object' || error === null) return false
  return (error as { response?: { status?: number } }).response?.status === status
}

/**
 * 增量计算文件 SHA-256（小写 hex）。
 *
 * 用 js-sha256 的 create()/update() 流式喂入 2 MiB 分块：每次只把一个分块读进内存，
 * 因此 1 GiB 文件也只会占 2 MiB 级别的堆（另有上游 Blob 读取的临时缓冲）。
 * 结果与 crypto.subtle.digest('SHA-256', 整个文件) 完全一致（e2e 用大文件哈希回读校验）。
 */
export async function sha256HexIncremental(
  file: File,
  onProgress?: (ratio: number) => void,
): Promise<string> {
  const hasher = sha256.create()
  for (let offset = 0; offset < file.size; offset += HASH_CHUNK_BYTES) {
    const end = Math.min(offset + HASH_CHUNK_BYTES, file.size)
    hasher.update(await file.slice(offset, end).arrayBuffer())
    onProgress?.(end / file.size)
  }
  return hasher.hex()
}

/**
 * 把预签名 URL 归一化为同源地址（仅当它指向本服务的 /v1/* 时）。
 *
 * 背景：local 驱动的预签名 URL 由后端配置 base_url 拼成绝对地址
 * （如 http://127.0.0.1:8099/v1/demo/objects/...）。浏览器从 :3000 打开页面时直连
 * 它属于跨源 PUT，会先发 CORS 预检；后端未挂 CORS 中间件，预检 404，直传必然失败。
 * 而 local 驱动的签名信息全在 query（token/expires）里，签名载荷只覆盖
 * bucket/key/op/expires/upload_id/part_number，**不含 host**，因此改写为同源相对路径
 * 后仍能通过服务端校验，并复用 Vite 的 /v1 代理（生产同源部署时行为不变）。
 *
 * 仅改写 pathname 以 /v1/ 开头的地址；S3/COS 等云存储的预签名 URL 不在 /v1/
 * 命名空间下且 host 参与签名，必须保持原样。
 */
export function toSameOriginURL(url: string): string {
  if (typeof window === 'undefined') return url
  const parsed = new URL(url, window.location.href)
  if (parsed.origin === window.location.origin) return url
  if (!parsed.pathname.startsWith('/v1/')) return url
  return `${parsed.pathname}${parsed.search}`
}

/** 上传单个分片，返回该分片的 ETag（complete 时原样回传）。 */
async function putPart(
  file: File,
  presigned: PresignURLResp,
  start: number,
  end: number,
): Promise<string> {
  // 预签名响应里的 headers 是签名覆盖的头（S3 场景漏发会 SignatureDoesNotMatch），
  // local 驱动为空表；这里逐值原样透传。
  const headers = new Headers()
  for (const [name, values] of Object.entries(presigned.headers ?? {})) {
    const list = Array.isArray(values) ? values : [String(values)]
    for (const value of list) headers.append(name, value)
  }

  const response = await fetch(toSameOriginURL(presigned.url), {
    method: presigned.method || 'PUT',
    headers,
    // 直接提交 Blob 分片：浏览器从磁盘按需读取，不会把整个文件读进内存
    body: file.slice(start, end),
  })
  const text = await response.text()
  if (!response.ok) {
    throw new Error(`分片上传失败（HTTP ${response.status}）：${text.slice(0, 200)}`)
  }

  let payload: { code?: number; msg?: string; data?: { etag?: string } }
  try {
    payload = JSON.parse(text) as typeof payload
  } catch {
    throw new Error(`分片上传返回非 JSON 响应：${text.slice(0, 200)}`)
  }
  if (payload.code !== 0) {
    throw new Error(payload.msg || `分片上传失败（code=${payload.code}）`)
  }
  // 响应体与 ETag 响应头都带分片 ETag（存储协议端点按 HTTP 语义返回），优先取响应体
  const etag = payload.data?.etag ?? response.headers.get('etag')?.replace(/"/g, '')
  if (!etag) throw new Error('分片上传响应缺少 etag，无法完成合并')
  return etag
}

/** 分片阶段的总体进度百分比（摘要阶段占用前 HASH_PROGRESS_WEIGHT%）。 */
function uploadPercent(done: number, total: number): number {
  return HASH_PROGRESS_WEIGHT + Math.round((done / total) * (100 - HASH_PROGRESS_WEIGHT))
}

/**
 * 以分片方式上传整个文件：增量摘要 → 创建会话 → 逐片预签名 + 直传 → complete。
 *
 * 任一步失败都会 abort 会话（尽力而为，清理已写入的分片），再把原始错误抛出；
 * 创建会话命中 409（同内容已存在）时抛 {@link MultipartContentExistsError}，
 * 此时没有会话需要清理。
 */
export async function uploadMultipartFile(
  file: File,
  onProgress?: (progress: MultipartProgress) => void,
): Promise<MultipartUploadResult> {
  const partsTotal = Math.max(1, Math.ceil(file.size / PART_SIZE_BYTES))

  const contentHash = await sha256HexIncremental(file, (ratio) => {
    onProgress?.({
      phase: 'hashing',
      percent: Math.round(ratio * HASH_PROGRESS_WEIGHT),
      partsDone: 0,
      partsTotal,
    })
  })

  let fileID: string | null = null
  try {
    const session = await createMultipartUpload({
      content_hash: contentHash,
      name: file.name,
      size: file.size,
      mime_type: file.type || 'application/octet-stream',
    }).catch((error: unknown) => {
      if (isHTTPStatus(error, 409)) throw new MultipartContentExistsError(contentHash)
      throw error
    })
    fileID = session.file_id

    const parts: MultipartPart[] = []
    for (let index = 0; index < partsTotal; index += 1) {
      const partNumber = index + 1
      const start = index * PART_SIZE_BYTES
      const end = Math.min(start + PART_SIZE_BYTES, file.size)
      onProgress?.({
        phase: 'uploading',
        percent: uploadPercent(index, partsTotal),
        partsDone: index,
        partsTotal,
      })
      const presigned = await presignUploadPart(session.file_id, partNumber)
      const etag = await putPart(file, presigned, start, end)
      parts.push({ part_number: partNumber, etag })
      onProgress?.({
        phase: 'uploading',
        percent: uploadPercent(partNumber, partsTotal),
        partsDone: partNumber,
        partsTotal,
      })
    }

    const record = await completeMultipartUpload(session.file_id, parts)
    onProgress?.({ phase: 'uploading', percent: 100, partsDone: partsTotal, partsTotal })
    return { file: record, contentHash, partsTotal }
  } catch (error) {
    if (fileID) {
      // 清理失败不应覆盖真正的失败原因，因此忽略 abort 自身的错误
      await abortMultipartUpload(fileID).catch(() => undefined)
    }
    throw error
  }
}
