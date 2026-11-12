import axios, { type AxiosRequestConfig } from 'axios'
import type {
  AffectedResult,
  ApiResponse,
  AsyncTask,
  AsyncTaskListResult,
  AsyncTaskRun,
  AsyncTaskRunListResult,
  CronTask,
  CronTaskListResult,
  CronTaskRun,
  CronTaskRunListResult,
  DbCheck,
  DictIntegrityReport,
  DictItemListResult,
  DictTreeNode,
  DictType,
  DictTypeCreateReq,
  DictTypeListResult,
  DictTypeUpdateReq,
  CheckExistResp,
  FileDetail,
  FileRecord,
  ListPartsResp,
  PageQuery,
  PageResult,
  PresignURLResp,
  ProductCreateReq,
  ProductDetail,
  ProductPageListItem,
  ProductUpdateReq,
  Setting,
  SettingCreateReq,
  SettingListResult,
  SettingUpdateReq,
  StatusUpdateReq,
} from '@ark-go-starter/types'

export const http = axios.create({
  baseURL: '/v1',
  timeout: 10000,
})

/** 取当前登录令牌；返回空值表示未登录（不注入 Authorization）。由宿主在 auth 层注入实现。 */
export type TokenProvider = () => string | null | undefined

/** 未认证（HTTP 401 或业务未认证码）回调；由宿主注入（如跳转 IAM 登录页）。 */
export type UnauthorizedHandler = () => void

let tokenProvider: TokenProvider | null = null
let unauthorizedHandler: UnauthorizedHandler | null = null

/**
 * 注册令牌提供者：request 拦截器据此注入 `Authorization: Bearer <token>`。
 * 传 null 解除注册。宿主（demo-web 的 auth 层）在登录态变化时调用。
 */
export function setTokenProvider(provider: TokenProvider | null): void {
  tokenProvider = provider
}

/** 注册未认证处理器：收到 401 时触发（通常跳转 IAM 登录页）。传 null 解除注册。 */
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  unauthorizedHandler = handler
}

http.interceptors.request.use((config) => {
  const token = tokenProvider?.()
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse
    if (body.code !== 0) {
      // 与后端鉴权一致：401 语义由宿主处理（跳登录），其余业务错误直接抛错
      if (response.status === 401) {
        unauthorizedHandler?.()
      }
      return Promise.reject(new Error(body.msg || `request failed, code=${body.code}`))
    }
    return body.data as unknown as typeof response
  },
  (error) => {
    // HTTP 层 401（中间件拒绝，非 body.code 路径）→ 触发未认证处理（跳 IAM 登录）
    if (error?.response?.status === 401) {
      unauthorizedHandler?.()
    }
    return Promise.reject(error)
  },
)

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await http.request<unknown, unknown>(config)
  return response as T
}

/** 后端列表分页参数：page + page_size（snake_case） */
interface BackendPageQuery {
  // 页码
  page?: number
  // 每页条数（后端下划线命名）
  page_size?: number
}

/** 把前端驼峰分页参数 { page, pageSize } 映射为后端 { page, page_size } */
function toPageQuery(params: PageQuery = {}): BackendPageQuery {
  const query: BackendPageQuery = {}
  if (params.page !== undefined) query.page = params.page
  if (params.pageSize !== undefined) query.page_size = params.pageSize
  return query
}

/** 合并分页参数与其它筛选参数（分页统一转成 page / page_size） */
function withPaging<T extends PageQuery>(params: T): Record<string, unknown> {
  const { page, pageSize, ...rest } = params
  return { ...rest, ...toPageQuery({ page, pageSize }) }
}

// ============================ 商品 Product ============================

/** 商品分页列表 GET /demo/products */
export function getProducts(params: PageQuery = {}) {
  return request<PageResult<ProductPageListItem>>({
    url: '/demo/products',
    method: 'get',
    params: toPageQuery(params),
  })
}

/** 商品详情 GET /demo/products/{productID} */
export function getProduct(id: number) {
  return request<ProductDetail>({
    url: `/demo/products/${id}`,
    method: 'get',
  })
}

/** 新建商品 POST /demo/products */
export function createProduct(body: ProductCreateReq) {
  return request<ProductDetail>({
    url: '/demo/products',
    method: 'post',
    data: body,
  })
}

/** 更新商品 PUT /demo/products/{productID} */
export function updateProduct(id: number, body: ProductUpdateReq) {
  return request<unknown>({
    url: `/demo/products/${id}`,
    method: 'put',
    data: body,
  })
}

/** 删除商品 DELETE /demo/products/{productID} */
export function deleteProduct(id: number) {
  return request<unknown>({
    url: `/demo/products/${id}`,
    method: 'delete',
  })
}

// ============================ 数据字典 Dict ============================

/** 字典类型分页列表 GET /demo/dict-types */
export function getDictTypes(params: PageQuery & { code?: string; status?: string } = {}) {
  return request<DictTypeListResult>({
    url: '/demo/dict-types',
    method: 'get',
    params: withPaging(params),
  })
}

/** 新建字典类型 POST /demo/dict-types */
export function createDictType(body: DictTypeCreateReq) {
  return request<DictType>({
    url: '/demo/dict-types',
    method: 'post',
    data: body,
  })
}

/** 更新字典类型 PUT /demo/dict-types/{id} */
export function updateDictType(id: string, body: DictTypeUpdateReq) {
  return request<unknown>({
    url: `/demo/dict-types/${id}`,
    method: 'put',
    data: body,
  })
}

/** 删除字典类型 DELETE /demo/dict-types/{id} */
export function deleteDictType(id: string) {
  return request<unknown>({
    url: `/demo/dict-types/${id}`,
    method: 'delete',
  })
}

/** 启停字典类型 PATCH /demo/dict-types/{id}/status */
export function updateDictTypeStatus(id: string, status: string) {
  return request<unknown>({
    url: `/demo/dict-types/${id}/status`,
    method: 'patch',
    data: { status } satisfies StatusUpdateReq,
  })
}

/** 字典项列表 GET /demo/dict-codes/{code}/items */
export function getDictItems(code: string, params: { includeDisabled?: boolean } = {}) {
  const query: Record<string, unknown> = {}
  if (params.includeDisabled !== undefined) query.include_disabled = params.includeDisabled
  return request<DictItemListResult>({
    url: `/demo/dict-codes/${code}/items`,
    method: 'get',
    params: query,
  })
}

/** 字典树完整性检查 GET /demo/dict-types/{id}/integrity-check（只报告结构性问题，不自动修复） */
export function checkDictTypeIntegrity(id: string) {
  return request<DictIntegrityReport>({
    url: `/demo/dict-types/${id}/integrity-check`,
    method: 'get',
  })
}

/** 字典项树 GET /demo/dict-codes/{code}/tree */
export function getDictTree(code: string, params: { includeDisabled?: boolean } = {}) {
  const query: Record<string, unknown> = {}
  if (params.includeDisabled !== undefined) query.include_disabled = params.includeDisabled
  return request<{ list: DictTreeNode[] }>({
    url: `/demo/dict-codes/${code}/tree`,
    method: 'get',
    params: query,
  })
}

// ============================ 配置中心 Setting ============================

/** 配置项分页列表 GET /demo/settings */
export function getSettings(
  params: PageQuery & { group?: string; key?: string; valueType?: string; status?: string } = {},
) {
  const { page, pageSize, valueType, ...rest } = params
  return request<SettingListResult>({
    url: '/demo/settings',
    method: 'get',
    params: { ...rest, ...(valueType !== undefined ? { value_type: valueType } : {}), ...toPageQuery({ page, pageSize }) },
  })
}

/** 新建配置项 POST /demo/settings */
export function createSetting(body: SettingCreateReq) {
  return request<Setting>({
    url: '/demo/settings',
    method: 'post',
    data: body,
  })
}

/** 更新配置项 PUT /demo/settings/{id} */
export function updateSetting(id: string, body: SettingUpdateReq) {
  return request<unknown>({
    url: `/demo/settings/${id}`,
    method: 'put',
    data: body,
  })
}

/** 删除配置项 DELETE /demo/settings/{id} */
export function deleteSetting(id: string) {
  return request<unknown>({
    url: `/demo/settings/${id}`,
    method: 'delete',
  })
}

// ============================ 定时任务 Cron ============================

/** 定时任务分页列表 GET /demo/cron-tasks */
export function getCronTasks(
  params: PageQuery & { taskType?: string; status?: string; bizType?: string; bizID?: string } = {},
) {
  const { page, pageSize, taskType, bizType, bizID, ...rest } = params
  return request<CronTaskListResult>({
    url: '/demo/cron-tasks',
    method: 'get',
    params: {
      ...rest,
      ...(taskType !== undefined ? { task_type: taskType } : {}),
      ...(bizType !== undefined ? { biz_type: bizType } : {}),
      ...(bizID !== undefined ? { biz_id: bizID } : {}),
      ...toPageQuery({ page, pageSize }),
    },
  })
}

/** 定时任务详情 GET /demo/cron-tasks/{id} */
export function getCronTask(id: string) {
  return request<CronTask>({
    url: `/demo/cron-tasks/${id}`,
    method: 'get',
  })
}

/** 启停定时任务 PATCH /demo/cron-tasks/{id}/status */
export function updateCronTaskStatus(id: string, status: string) {
  return request<unknown>({
    url: `/demo/cron-tasks/${id}/status`,
    method: 'patch',
    data: { status } satisfies StatusUpdateReq,
  })
}

/** 删除定时任务 DELETE /demo/cron-tasks/{id} */
export function deleteCronTask(id: string) {
  return request<unknown>({
    url: `/demo/cron-tasks/${id}`,
    method: 'delete',
  })
}

/** 定时任务运行记录分页列表 GET /demo/cron-tasks/{id}/runs */
export function getCronTaskRuns(id: string, params: PageQuery & { status?: string } = {}) {
  return request<CronTaskRunListResult>({
    url: `/demo/cron-tasks/${id}/runs`,
    method: 'get',
    params: withPaging(params),
  })
}

/** 定时任务运行记录详情 GET /demo/cron-runs/{id} */
export function getCronRun(id: string) {
  return request<CronTaskRun>({
    url: `/demo/cron-runs/${id}`,
    method: 'get',
  })
}

/**
 * 标记某任务的陈旧 running 运行记录为 failed POST /demo/cron-tasks/{id}/runs/mark-stale。
 *
 * 契约（golib v1.34.1 gincron）：body 为 `{ cutoff_seconds }`（必填且 > 0），
 * 语义是「start_at 早于 now-cutoff 且仍为 running 的记录视为陈旧」。
 */
export function markStaleCronTaskRuns(id: string, cutoffSeconds: number) {
  return request<AffectedResult>({
    url: `/demo/cron-tasks/${id}/runs/mark-stale`,
    method: 'post',
    data: { cutoff_seconds: cutoffSeconds } satisfies { cutoff_seconds: number },
  })
}

/**
 * 清理某任务 before 之前的运行记录 DELETE /demo/cron-tasks/{id}/runs。
 *
 * 注意：不是 query 参数 `?before=`，golib 契约要求 JSON body `{ before_unix }`（unix 秒，必填且 > 0），
 * 缺失时后端返回 "invalid request: ..." 绑定错误。
 */
export function cleanupCronTaskRuns(id: string, beforeUnix: number) {
  return request<AffectedResult>({
    url: `/demo/cron-tasks/${id}/runs`,
    method: 'delete',
    data: { before_unix: beforeUnix } satisfies { before_unix: number },
  })
}

// ============================ 异步任务 Async ============================

/** 异步任务分页列表 GET /demo/async-tasks */
export function getAsyncTasks(params: PageQuery & { status?: string } = {}) {
  return request<AsyncTaskListResult>({
    url: '/demo/async-tasks',
    method: 'get',
    params: withPaging(params),
  })
}

/** 异步任务详情（按类型） GET /demo/async-tasks/{type} */
export function getAsyncTask(type: string) {
  return request<AsyncTask>({
    url: `/demo/async-tasks/${type}`,
    method: 'get',
  })
}

/** 启停异步任务 PATCH /demo/async-tasks/{type}/status */
export function updateAsyncTaskStatus(type: string, status: string) {
  return request<unknown>({
    url: `/demo/async-tasks/${type}/status`,
    method: 'patch',
    data: { status } satisfies StatusUpdateReq,
  })
}

/** 异步任务运行记录分页列表 GET /demo/async-tasks/{type}/runs */
export function getAsyncTaskRuns(type: string, params: PageQuery & { queue?: string; status?: string } = {}) {
  return request<AsyncTaskRunListResult>({
    url: `/demo/async-tasks/${type}/runs`,
    method: 'get',
    params: withPaging(params),
  })
}

/** 异步任务运行记录详情 GET /demo/async-runs/{id} */
export function getAsyncRun(id: string) {
  return request<AsyncTaskRun>({
    url: `/demo/async-runs/${id}`,
    method: 'get',
  })
}

/**
 * 标记某任务类型的陈旧 processing 运行记录为 failed POST /demo/async-tasks/{type}/runs/mark-stale。
 *
 * 契约（golib v1.34.1 ginasync）：body 为 `{ cutoff_seconds }`（必填且 > 0）。
 */
export function markStaleAsyncTaskRuns(type: string, cutoffSeconds: number) {
  return request<AffectedResult>({
    url: `/demo/async-tasks/${type}/runs/mark-stale`,
    method: 'post',
    data: { cutoff_seconds: cutoffSeconds } satisfies { cutoff_seconds: number },
  })
}

/** 清理某任务类型 before 之前的运行记录 DELETE /demo/async-tasks/{type}/runs（body `{ before_unix }`） */
export function cleanupAsyncTaskRuns(type: string, beforeUnix: number) {
  return request<AffectedResult>({
    url: `/demo/async-tasks/${type}/runs`,
    method: 'delete',
    data: { before_unix: beforeUnix } satisfies { before_unix: number },
  })
}

// ============================ 文件上传 Files ============================

/**
 * 上传单个文件 POST /demo/files（multipart，字段 file / content_hash）。
 *
 * contentHash 必填：后端 UploadRequest 上是 binding:"required"，缺失时接口稳定返回
 * {"code":-1,"msg":"content_hash is required"}；而且当传入值恰好是 64 位 hex 时，
 * filestore 会校验它与实际内容一致（不一致返回 content hash mismatch）。
 * 因此调用方必须传真实 SHA-256（见 demo-web 的 Files 页 sha256Hex）。
 *
 * 单独放宽超时（全局 10s 对大文件不够）：上传超时会让浏览器中断请求，
 * 后端可能已落盘但前端拿不到结果，形成"看起来失败、实际成功"的脏状态。
 */
export function uploadFile(file: File, contentHash: string) {
  const form = new FormData()
  form.append('file', file)
  form.append('content_hash', contentHash)
  return request<FileRecord>({
    url: '/demo/files',
    method: 'post',
    data: form,
    timeout: 120000,
    // Content-Type 交给浏览器自动生成（含 multipart boundary），此处不手动设置
  })
}

/** 文件记录（上传成功后返回的窄字段） GET /demo/files/{id} 的旧调用点保留 */
export function getFile(id: string) {
  return request<FileRecord>({
    url: `/demo/files/${id}`,
    method: 'get',
  })
}

/** 文件完整详情 GET /demo/files/{id}（含 content_hash/size/storage_uri/时间等全量字段） */
export function getFileDetail(id: string) {
  return request<FileDetail>({
    url: `/demo/files/${id}`,
    method: 'get',
  })
}

/**
 * 列举分片会话已上传分片 GET /demo/files/{id}/parts。
 * 仅分片上传（> 8 MB）产生的文件有意义；普通上传/已合并后可能为空或报错，属正常。
 * maxParts / partNumberMarker 为可选分页参数（后端 snake_case）。
 */
export function listFileParts(
  id: string,
  params: { maxParts?: number; partNumberMarker?: number } = {},
) {
  const query: Record<string, unknown> = {}
  if (params.maxParts !== undefined) query.max_parts = params.maxParts
  if (params.partNumberMarker !== undefined) query.part_number_marker = params.partNumberMarker
  return request<ListPartsResp>({
    url: `/demo/files/${id}/parts`,
    method: 'get',
    params: query,
  })
}

// ---------------------------- serve / redirect URL 拼装 ----------------------------
//
// 这几条不是业务 JSON 接口，而是「浏览器直接导航/下载」的端点（serve 直出文件流、redirect 302 跳转）。
// 因此不做 axios 请求，只拼出同源相对路径交给 <a download> / window.open。
// query 形式兼容 file_id 或 storage_uri（见 ginupload getFileQueryRequest）。

/** serve 直出（路径参数） URL：GET /demo/files/{id}/serve（仅 local 驱动有效） */
export function serveFileURL(id: string): string {
  return `/v1/demo/files/${id}/serve`
}

/** redirect 302（路径参数） URL：GET /demo/files/{id}/redirect */
export function redirectFileURL(id: string): string {
  return `/v1/demo/files/${id}/redirect`
}

/** serve 直出（query 形式） URL：GET /demo/files/serve?file_id= 或 ?storage_uri= */
export function serveFileURLByQuery(q: { fileId?: string; storageUri?: string }): string {
  const search = new URLSearchParams()
  if (q.fileId) search.set('file_id', q.fileId)
  if (q.storageUri) search.set('storage_uri', q.storageUri)
  return `/v1/demo/files/serve?${search.toString()}`
}

/** redirect 302（query 形式） URL：GET /demo/files/redirect?file_id= 或 ?storage_uri= */
export function redirectFileURLByQuery(q: { fileId?: string; storageUri?: string }): string {
  const search = new URLSearchParams()
  if (q.fileId) search.set('file_id', q.fileId)
  if (q.storageUri) search.set('storage_uri', q.storageUri)
  return `/v1/demo/files/redirect?${search.toString()}`
}

/** 删除文件 DELETE /demo/files/{id} */
export function deleteFile(id: string) {
  return request<unknown>({
    url: `/demo/files/${id}`,
    method: 'delete',
  })
}

/** 获取下载地址 POST /demo/files/{id}/presign-url */
export function presignDownload(id: string) {
  return request<PresignURLResp>({
    url: `/demo/files/${id}/presign-url`,
    method: 'post',
  })
}

// ---------------------------- 分片上传（multipart） ----------------------------
//
// 契约来源：golib biz/gserver/ginupload router.go 的 /files/multipart 子资源。
// 流程：创建会话 → 逐片取预签名地址并直传 → complete 合并；失败用 abort 清理。
// 与普通上传的关键差异：create 时必须带**整个文件**的 SHA-256（后端按它去重），
// 因此调用方需要增量计算摘要，不能一次性 arrayBuffer（见 demo-web Files 页）。

/** 分片上传创建请求体（POST /demo/files/multipart，字段均必填） */
export interface MultipartCreateReq {
  // 整个文件的 SHA-256（小写 hex），后端据此做内容去重；同内容已存在时返回 HTTP 409
  content_hash: string
  // 文件名
  name: string
  // 文件大小（字节）；complete 时后端会与实际合并结果对账，不一致报 size mismatch
  size: number
  // MIME 类型（可空，缺省 application/octet-stream）
  mime_type?: string
}

/** 分片上传会话（createMultipartUpload 返回） */
export interface MultipartCreateResp {
  // 文件 ID；parts/complete/abort 的路径参数都用它
  file_id: string
  // 存储侧上传会话 ID（local 驱动为分片目录名，仅诊断用）
  upload_id: string
}

/** 单个分片信息：complete 时回传，etag 来自分片 PUT 的响应 */
export interface MultipartPart {
  // 分片编号，从 1 开始；complete 时必须严格升序且不重复
  part_number: number
  // 分片内容 ETag（local 驱动为内容 MD5，S3 为上传响应 ETag），用分片 PUT 的返回值
  etag: string
}

/** 创建分片上传会话 POST /demo/files/multipart */
export function createMultipartUpload(req: MultipartCreateReq) {
  return request<MultipartCreateResp>({
    url: '/demo/files/multipart',
    method: 'post',
    data: req,
  })
}

/**
 * 获取单个分片的预签名上传地址 POST /demo/files/multipart/{fileID}/parts。
 *
 * 返回的 url/method/headers 必须原样使用：headers 是签名覆盖的头（S3 场景下漏发
 * 会得到 SignatureDoesNotMatch）。本地驱动的 url 是配置项 base_url 拼出的后端绝对
 * 地址，浏览器跨源直传会被 CORS 预检拦下；demo-web 侧会把它改写成同源相对路径再
 * PUT（见 pages/Files/multipart.ts 的 toSameOriginURL 说明）。
 */
export function presignUploadPart(fileID: string, partNumber: number) {
  return request<PresignURLResp>({
    url: `/demo/files/multipart/${fileID}/parts`,
    method: 'post',
    data: { part_number: partNumber },
  })
}

/** 完成分片上传（服务端按 parts 顺序合并） POST /demo/files/multipart/{fileID}/complete */
export function completeMultipartUpload(fileID: string, parts: MultipartPart[]) {
  return request<FileRecord>({
    url: `/demo/files/multipart/${fileID}/complete`,
    method: 'post',
    data: { parts },
    // 服务端要把所有分片合并成最终对象，超大文件合并耗时可能超过全局 10s 超时
    timeout: 120000,
  })
}

/**
 * 取消分片上传并清理已写入的分片 DELETE /demo/files/multipart/{fileID}。
 * 仅在上传失败/用户取消时调用；complete 成功后会话已被服务端清理，再 abort 会报错。
 */
export function abortMultipartUpload(fileID: string) {
  return request<unknown>({
    url: `/demo/files/multipart/${fileID}`,
    method: 'delete',
  })
}

/** 按内容哈希检查文件是否已存在（秒传探测） POST /demo/files/check-exist */
export function checkFileExists(contentHash: string) {
  return request<CheckExistResp>({
    url: '/demo/files/check-exist',
    method: 'post',
    data: { content_hash: contentHash },
  })
}

// ============================ 健康检查 Health ============================

/** 健康检查 GET /demo/health，返回各依赖连通性 */
export function getHealth() {
  return request<DbCheck>({
    url: '/demo/health',
    method: 'get',
  })
}
