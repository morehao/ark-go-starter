import axios, { type AxiosRequestConfig } from 'axios'
import type {
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
  DictItemListResult,
  DictTreeNode,
  DictType,
  DictTypeCreateReq,
  DictTypeListResult,
  DictTypeUpdateReq,
  FileRecord,
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

/** 字典类型分页列表 GET /dict-types */
export function getDictTypes(params: PageQuery & { code?: string; status?: string } = {}) {
  return request<DictTypeListResult>({
    url: '/dict-types',
    method: 'get',
    params: withPaging(params),
  })
}

/** 新建字典类型 POST /dict-types */
export function createDictType(body: DictTypeCreateReq) {
  return request<DictType>({
    url: '/dict-types',
    method: 'post',
    data: body,
  })
}

/** 更新字典类型 PUT /dict-types/{id} */
export function updateDictType(id: string, body: DictTypeUpdateReq) {
  return request<unknown>({
    url: `/dict-types/${id}`,
    method: 'put',
    data: body,
  })
}

/** 删除字典类型 DELETE /dict-types/{id} */
export function deleteDictType(id: string) {
  return request<unknown>({
    url: `/dict-types/${id}`,
    method: 'delete',
  })
}

/** 启停字典类型 PATCH /dict-types/{id}/status */
export function updateDictTypeStatus(id: string, status: string) {
  return request<unknown>({
    url: `/dict-types/${id}/status`,
    method: 'patch',
    data: { status } satisfies StatusUpdateReq,
  })
}

/** 字典项列表 GET /dict-codes/{code}/items */
export function getDictItems(code: string, params: { includeDisabled?: boolean } = {}) {
  const query: Record<string, unknown> = {}
  if (params.includeDisabled !== undefined) query.include_disabled = params.includeDisabled
  return request<DictItemListResult>({
    url: `/dict-codes/${code}/items`,
    method: 'get',
    params: query,
  })
}

/** 字典项树 GET /dict-codes/{code}/tree */
export function getDictTree(code: string, params: { includeDisabled?: boolean } = {}) {
  const query: Record<string, unknown> = {}
  if (params.includeDisabled !== undefined) query.include_disabled = params.includeDisabled
  return request<{ list: DictTreeNode[] }>({
    url: `/dict-codes/${code}/tree`,
    method: 'get',
    params: query,
  })
}

// ============================ 配置中心 Setting ============================

/** 配置项分页列表 GET /settings */
export function getSettings(
  params: PageQuery & { group?: string; key?: string; valueType?: string; status?: string } = {},
) {
  const { page, pageSize, valueType, ...rest } = params
  return request<SettingListResult>({
    url: '/settings',
    method: 'get',
    params: { ...rest, ...(valueType !== undefined ? { value_type: valueType } : {}), ...toPageQuery({ page, pageSize }) },
  })
}

/** 新建配置项 POST /settings */
export function createSetting(body: SettingCreateReq) {
  return request<Setting>({
    url: '/settings',
    method: 'post',
    data: body,
  })
}

/** 更新配置项 PUT /settings/{id} */
export function updateSetting(id: string, body: SettingUpdateReq) {
  return request<unknown>({
    url: `/settings/${id}`,
    method: 'put',
    data: body,
  })
}

/** 删除配置项 DELETE /settings/{id} */
export function deleteSetting(id: string) {
  return request<unknown>({
    url: `/settings/${id}`,
    method: 'delete',
  })
}

// ============================ 定时任务 Cron ============================

/** 定时任务分页列表 GET /cron-tasks */
export function getCronTasks(
  params: PageQuery & { taskType?: string; status?: string; bizType?: string; bizID?: string } = {},
) {
  const { page, pageSize, taskType, bizType, bizID, ...rest } = params
  return request<CronTaskListResult>({
    url: '/cron-tasks',
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

/** 定时任务详情 GET /cron-tasks/{id} */
export function getCronTask(id: string) {
  return request<CronTask>({
    url: `/cron-tasks/${id}`,
    method: 'get',
  })
}

/** 启停定时任务 PATCH /cron-tasks/{id}/status */
export function updateCronTaskStatus(id: string, status: string) {
  return request<unknown>({
    url: `/cron-tasks/${id}/status`,
    method: 'patch',
    data: { status } satisfies StatusUpdateReq,
  })
}

/** 删除定时任务 DELETE /cron-tasks/{id} */
export function deleteCronTask(id: string) {
  return request<unknown>({
    url: `/cron-tasks/${id}`,
    method: 'delete',
  })
}

/** 定时任务运行记录分页列表 GET /cron-tasks/{id}/runs */
export function getCronTaskRuns(id: string, params: PageQuery & { status?: string } = {}) {
  return request<CronTaskRunListResult>({
    url: `/cron-tasks/${id}/runs`,
    method: 'get',
    params: withPaging(params),
  })
}

/** 定时任务运行记录详情 GET /cron-runs/{id} */
export function getCronRun(id: string) {
  return request<CronTaskRun>({
    url: `/cron-runs/${id}`,
    method: 'get',
  })
}

// ============================ 异步任务 Async ============================

/** 异步任务分页列表 GET /async-tasks */
export function getAsyncTasks(params: PageQuery & { status?: string } = {}) {
  return request<AsyncTaskListResult>({
    url: '/async-tasks',
    method: 'get',
    params: withPaging(params),
  })
}

/** 异步任务详情（按类型） GET /async-tasks/{type} */
export function getAsyncTask(type: string) {
  return request<AsyncTask>({
    url: `/async-tasks/${type}`,
    method: 'get',
  })
}

/** 启停异步任务 PATCH /async-tasks/{type}/status */
export function updateAsyncTaskStatus(type: string, status: string) {
  return request<unknown>({
    url: `/async-tasks/${type}/status`,
    method: 'patch',
    data: { status } satisfies StatusUpdateReq,
  })
}

/** 异步任务运行记录分页列表 GET /async-tasks/{type}/runs */
export function getAsyncTaskRuns(type: string, params: PageQuery & { queue?: string; status?: string } = {}) {
  return request<AsyncTaskRunListResult>({
    url: `/async-tasks/${type}/runs`,
    method: 'get',
    params: withPaging(params),
  })
}

/** 异步任务运行记录详情 GET /async-runs/{id} */
export function getAsyncRun(id: string) {
  return request<AsyncTaskRun>({
    url: `/async-runs/${id}`,
    method: 'get',
  })
}

// ============================ 文件上传 Files ============================

/** 上传单个文件 POST /demo/files（multipart，字段 file / content_hash） */
export function uploadFile(file: File, contentHash?: string) {
  const form = new FormData()
  form.append('file', file)
  if (contentHash) form.append('content_hash', contentHash)
  return request<FileRecord>({
    url: '/demo/files',
    method: 'post',
    data: form,
    // Content-Type 交给浏览器自动生成（含 multipart boundary），此处不手动设置
  })
}

/** 文件详情 GET /demo/files/{id} */
export function getFile(id: string) {
  return request<FileRecord>({
    url: `/demo/files/${id}`,
    method: 'get',
  })
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

// ============================ 健康检查 Health ============================

/** 健康检查 GET /demo/health，返回各依赖连通性 */
export function getHealth() {
  return request<DbCheck>({
    url: '/demo/health',
    method: 'get',
  })
}
