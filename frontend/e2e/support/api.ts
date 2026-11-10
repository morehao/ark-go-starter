/**
 * 直连后端的 REST 辅助，用于造数与清理。
 *
 * e2e 数据统一用 e2e- 前缀 + 时间戳，保证唯一；清理时按前缀或 id 删除，避免污染真实库。
 * baseURL 指向后端根（默认 http://localhost:8099），不经过前端代理，减少一层依赖。
 *
 * 注意路径前缀：demo 后端用 ginserver 把全部业务模块挂在 `/v1/demo/*` 下，
 * 因此这里所有路径都带 `/v1/demo`。
 */
export const API_BASE = process.env.E2E_API_BASE ?? 'http://localhost:8099'

export interface ProductCreateReq {
  name: string
  categoryID: number
  price: number
}

export interface ProductItem {
  productID: number
  name: string
  categoryID: number
  price: number
}

interface ApiResp<T> {
  code: number
  msg: string
  data: T
}

/** 生成唯一测试名，前缀 e2e-。 */
export function uniqueName(prefix = 'e2e-product'): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e4)}`
}

async function call<T>(method: string, path: string, body?: unknown): Promise<ApiResp<T>> {
  const resp = await fetch(`${API_BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await resp.text()
  if (!resp.ok) {
    throw new Error(`${method} ${path} -> HTTP ${resp.status}: ${text}`)
  }
  return JSON.parse(text) as ApiResp<T>
}

/** 调用接口并断言业务码为 0（失败消息带上下文，便于定位）。 */
async function callOK<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await call<T>(method, path, body)
  if (res.code !== 0) throw new Error(`${method} ${path} -> code=${res.code}: ${res.msg}`)
  return res.data
}

// ============================ 商品 Product ============================

/** 通过接口创建商品，返回商品 ID。 */
export async function createProduct(req: ProductCreateReq): Promise<number> {
  const res = await call<{ productID: number }>('POST', '/v1/demo/products', req)
  if (res.code !== 0) throw new Error(`create product failed: ${res.msg}`)
  return res.data.productID
}

/** 通过接口删除商品（软删）。 */
export async function deleteProduct(id: number): Promise<void> {
  await call<unknown>('DELETE', `/v1/demo/products/${id}`)
}

/** 列出商品（分页）。 */
export async function listProducts(page = 1, pageSize = 50): Promise<{ list: ProductItem[]; total: number }> {
  const res = await call<{ list: ProductItem[]; total: number }>(
    'GET',
    `/v1/demo/products?page=${page}&pageSize=${pageSize}`,
  )
  return res.data
}

/** 清理所有名称以 e2e- 开头的商品（软删），用于 globalTeardown。 */
export async function cleanupE2EProducts(): Promise<void> {
  const { list } = await listProducts(1, 100)
  const targets = list.filter((p) => p.name.startsWith('e2e-'))
  await Promise.all(targets.map((p) => deleteProduct(p.productID).catch(() => undefined)))
}

// ============================ 数据字典 Dict ============================

export interface DictTypeItem {
  id: string
  code: string
  name: string
  status: string
  description: string
  created_at: number
}

export interface DictItemRec {
  id: string
  type_code: string
  value: string
  label: string
  parent_id: string
  level: number
  sort: number
  status: string
}

export interface DictTypeCreateReq {
  code: string
  // 后端 code 必填、name 可选（空名时列表以 code 兜底展示）
  name?: string
  status?: string
  description?: string
}

export interface DictItemCreateReq {
  type_code: string
  value: string
  label: string
  parent_id?: string
  sort?: number
  status?: string
  description?: string
}

/** 新建字典类型，返回后端完整记录（含 id）。 */
export async function createDictType(req: DictTypeCreateReq): Promise<DictTypeItem> {
  const data = await callOK<DictTypeItem>('POST', '/v1/demo/dict-types', {
    status: 'enabled',
    ...req,
  })
  // 后端 createType 返回完整实体；兜底：个别版本只回 id 时按 code 回查。
  if (data?.id) return data
  const found = (await listDictTypes(1, 100)).list.find((t) => t.code === req.code)
  if (!found) throw new Error(`create dict type ${req.code} but cannot find it`)
  return found
}

/** 新建字典项，返回后端完整记录（含 id/level）。 */
export async function createDictItem(req: DictItemCreateReq): Promise<DictItemRec> {
  return callOK<DictItemRec>('POST', '/v1/demo/dict-items', { status: 'enabled', ...req })
}

/** 字典类型分页列表。 */
export async function listDictTypes(
  page = 1,
  pageSize = 100,
): Promise<{ list: DictTypeItem[]; total: number }> {
  return callOK<{ list: DictTypeItem[]; total: number }>(
    'GET',
    `/v1/demo/dict-types?page=${page}&page_size=${pageSize}`,
  )
}

/** 删除字典类型；cascade=true 会级联删除其下全部字典项。 */
export async function deleteDictType(id: string, cascade = false): Promise<void> {
  await call<unknown>('DELETE', `/v1/demo/dict-types/${id}${cascade ? '?cascade=true' : ''}`)
}

/** 清理全部 e2e- 前缀字典类型（级联删项），用于 globalTeardown 与用例兜底。 */
export async function cleanupE2EDicts(): Promise<void> {
  const { list } = await listDictTypes(1, 100)
  const targets = list.filter((t) => t.code.startsWith('e2e-'))
  await Promise.all(targets.map((t) => deleteDictType(t.id, true).catch(() => undefined)))
}

/** 字典树巡检结果（对应 gindict.integrityResponse），供 UI 断言交叉复核。 */
export interface DictIntegrityReport {
  type_code: string
  checked: number
  healthy: boolean
  issues: Array<{ item_id: string; value: string; kind: string; detail: string }>
  orphans: string[]
}

/** 调完整性检查接口，用于复核 UI 抽屉里展示的数字。 */
export async function checkDictTypeIntegrityAPI(id: string): Promise<DictIntegrityReport> {
  return callOK<DictIntegrityReport>('GET', `/v1/demo/dict-types/${id}/integrity-check`)
}

// ============================ 配置中心 Setting ============================

export interface SettingRec {
  id: string
  group: string
  key: string
  value_type: string
  value: string
  encryption_mode: string
  description: string
  status: string
}

export interface SettingCreateReq {
  group: string
  key: string
  value_type?: string
  value: string
  description?: string
  // 是否加密落库（开启后响应 encryption_mode=encrypted）
  encrypted?: boolean
  // 状态，默认 enabled；传 disabled 可构造禁用项
  status?: string
}

/** 新建配置项（同 group+key 会 upsert，改值），返回列表中的完整记录。 */
export async function createSettingAPI(req: SettingCreateReq): Promise<SettingRec> {
  await callOK<unknown>('POST', '/v1/demo/settings', {
    value_type: 'string',
    ...req,
  })
  const found = (await listSettings(1, 100)).list.find(
    (s) => s.group === req.group && s.key === req.key,
  )
  if (!found) throw new Error(`create setting ${req.group}/${req.key} but cannot find it`)
  return found
}

/** 配置项分页列表。 */
export async function listSettings(
  page = 1,
  pageSize = 100,
): Promise<{ list: SettingRec[]; total: number }> {
  return callOK<{ list: SettingRec[]; total: number }>(
    'GET',
    `/v1/demo/settings?page=${page}&page_size=${pageSize}`,
  )
}

/** 删除配置项。 */
export async function deleteSettingAPI(id: string): Promise<void> {
  await call<unknown>('DELETE', `/v1/demo/settings/${id}`)
}

/** 清理全部 e2e- 前缀分组下的配置项。 */
export async function cleanupE2ESettings(prefix = 'e2e-'): Promise<void> {
  const { list } = await listSettings(1, 100)
  const targets = list.filter((s) => s.group.startsWith(prefix) || s.key.startsWith(prefix))
  await Promise.all(targets.map((s) => deleteSettingAPI(s.id).catch(() => undefined)))
}

// ============================ 定时任务 Cron / 异步任务 Async ============================
// 任务定义只能由后端代码注册（无 POST 建任务接口），因此这里只提供运行记录读取，
// 用于交叉复核 UI 上的 mark-stale / cleanup 动作没有误删或误标数据。

export interface CronTaskRunRec {
  id: string
  task_id: string
  start_at: number
  end_at?: number | null
  status: string
}

export interface AsyncTaskRunRec {
  id: string
  task_type: string
  queue: string
  status: string
}

export interface CronTaskRec {
  id: string
  name: string
  spec: string
  status: string
}

export interface AsyncTaskRec {
  id: string
  name: string
  status: string
}

/** 定时任务分页列表（任务由后端代码注册，这里只读以拿到 task id）。 */
export async function listCronTasks(): Promise<{ list: CronTaskRec[]; total: number }> {
  return callOK<{ list: CronTaskRec[]; total: number }>(
    'GET',
    '/v1/demo/cron-tasks?page=1&page_size=100',
  )
}

/** 异步任务分页列表（同上，只读）。 */
export async function listAsyncTasks(): Promise<{ list: AsyncTaskRec[]; total: number }> {
  return callOK<{ list: AsyncTaskRec[]; total: number }>(
    'GET',
    '/v1/demo/async-tasks?page=1&page_size=100',
  )
}

/** 定时任务运行记录分页列表。 */
export async function listCronTaskRuns(
  taskId: string,
  page = 1,
  pageSize = 100,
): Promise<{ list: CronTaskRunRec[]; total: number }> {
  return callOK<{ list: CronTaskRunRec[]; total: number }>(
    'GET',
    `/v1/demo/cron-tasks/${taskId}/runs?page=${page}&page_size=${pageSize}`,
  )
}

/** 异步任务运行记录分页列表。 */
export async function listAsyncTaskRuns(
  type: string,
  page = 1,
  pageSize = 100,
): Promise<{ list: AsyncTaskRunRec[]; total: number }> {
  return callOK<{ list: AsyncTaskRunRec[]; total: number }>(
    'GET',
    `/v1/demo/async-tasks/${type}/runs?page=${page}&page_size=${pageSize}`,
  )
}

// ============================ 文件 Files ============================

/** 删除文件。 */
export async function deleteFile(id: string): Promise<void> {
  await call<unknown>('DELETE', `/v1/demo/files/${id}`)
}

/** 秒传探测返回的文件记录（golib v1.34.1 起 exists=true 时返回完整字段）。 */
export interface FileExistFile {
  file_id: string
  name: string
  mime_type: string
  status: string
}

/** 按内容哈希检查是否已存在（秒传探测）。 */
export async function checkFileExists(
  contentHash: string,
): Promise<{ exists: boolean; file?: FileExistFile }> {
  return callOK<{ exists: boolean; file?: FileExistFile }>(
    'POST',
    '/v1/demo/files/check-exist',
    { content_hash: contentHash },
  )
}

/** 直接拉取文件内容（/serve 直出）：返回状态码、Content-Type 与文本内容。 */
export async function serveFile(
  id: string,
): Promise<{ status: number; contentType: string; text: string }> {
  const resp = await fetch(`${API_BASE}/v1/demo/files/${id}/serve`)
  return {
    status: resp.status,
    contentType: resp.headers.get('content-type') ?? '',
    text: await resp.text(),
  }
}
