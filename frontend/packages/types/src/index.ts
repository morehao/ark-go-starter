/** 通用响应包装：后端统一返回 { code, requestID, data, msg }，code === 0 表示成功 */
export interface ApiResponse<T = unknown> {
  // 业务状态码，0 为成功
  code: number
  // 请求链路 ID
  requestID: string
  // 业务数据
  data: T
  // 提示信息
  msg: string
}

/** 前端分页查询参数（驼峰，面向页面组件，内部由 api 层转成后端的 page / page_size） */
export interface PageQuery {
  // 页码，从 1 开始
  page?: number
  // 每页条数
  pageSize?: number
}

/** 分页结果：list 为当前页数据，total 为总条数 */
export interface PageResult<T> {
  // 当前页数据列表
  list: T[]
  // 数据总条数
  total: number
}

// ============================ 商品 Product ============================
// 示例业务对象 product，接口路径 /v1/demo/products，响应字段为驼峰。

/** 商品列表项（对应后端 ProductPageListItem） */
export interface ProductPageListItem {
  // 自增 ID
  productID: number
  // 商品名称
  name?: string
  // 分类 ID
  categoryID?: number
  // 价格
  price?: number
  // 创建人 ID
  createdBy?: number
  // 更新人 ID
  updatedBy?: number
  // 创建时间（unix 秒）
  createdAt?: number
  // 更新时间（unix 秒）
  updatedAt?: number
}

/** 商品详情（与列表项形状一致） */
export type ProductDetail = ProductPageListItem

/** 新建商品请求体 */
export interface ProductCreateReq {
  // 商品名称（必填）
  name: string
  // 分类 ID
  categoryID?: number
  // 价格
  price?: number
}

/** 更新商品请求体（局部更新，未传字段不修改） */
export interface ProductUpdateReq {
  // 商品名称
  name?: string
  // 分类 ID
  categoryID?: number
  // 价格
  price?: number
}

// ============================ 数据字典 Dict ============================
// golib gindict 路由，响应字段为 snake_case。

/** 字典类型 */
export interface DictType {
  // 类型 ID
  id: string
  // 类型编码（全表唯一，创建后不可变）
  code: string
  // 类型名称
  name: string
  // 状态：enabled / disabled
  status: string
  // 类型级扩展 JSON（字符串）
  extra?: string
  // 说明
  description?: string
  // 创建时间（unix 秒）
  created_at: number
  // 更新时间（unix 秒）
  updated_at: number
}

/** 字典项 */
export interface DictItem {
  // 项 ID
  id: string
  // 所属类型 code
  type_code: string
  // 码值（同类型内唯一）
  value: string
  // 展示文案
  label: string
  // 父项 ID，根为空串
  parent_id: string
  // 层级
  level: number
  // 同层排序
  sort: number
  // 状态：enabled / disabled
  status: string
  // 项级扩展 JSON（字符串）
  extra?: string
  // 说明
  description?: string
  // 创建时间（unix 秒）
  created_at: number
  // 更新时间（unix 秒）
  updated_at: number
}

/** 字典树节点：项字段平铺 + children，orphan 为孤儿节点标记 */
export interface DictTreeNode extends DictItem {
  // 子节点
  children?: DictTreeNode[]
  // 是否为孤儿节点（父节点缺失）
  orphan?: boolean
}

/** 字典类型分页结果 */
export interface DictTypeListResult {
  // 当前页类型列表
  list: DictType[]
  // 总条数
  total: number
}

/** 字典项列表结果（不分页） */
export interface DictItemListResult {
  // 项列表
  list: DictItem[]
}

/** 新建字典类型请求体 */
export interface DictTypeCreateReq {
  // 类型编码（必填）
  code: string
  // 类型名称
  name?: string
  // 状态，默认 enabled
  status?: string
  // 类型级扩展 JSON
  extra?: string
  // 说明
  description?: string
}

/** 更新字典类型请求体（局部更新） */
export interface DictTypeUpdateReq {
  // 类型名称
  name?: string
  // 类型级扩展 JSON
  extra?: string
  // 说明
  description?: string
}

/** 启停请求体：enabled / disabled */
export interface StatusUpdateReq {
  // 目标状态
  status: string
}

// ============================ 配置中心 Setting ============================
// golib ginsetting 路由，响应字段为 snake_case。

/** 配置项 */
export interface Setting {
  // 配置项 ID
  id: string
  // 配置分组
  group: string
  // 配置键名（同组内唯一）
  key: string
  // 值类型：json/toml/yaml/string/int/bool/float
  value_type: string
  // 配置值（明文）
  value: string
  // 加密模式：plain / encrypted
  encryption_mode: string
  // 描述
  description: string
  // 状态：enabled / disabled
  status: string
  // 创建时间（unix 秒）
  created_at: number
  // 更新时间（unix 秒）
  updated_at: number
}

/** 配置项分页结果 */
export interface SettingListResult {
  // 当前页配置项
  list: Setting[]
  // 总条数
  total: number
}

/** 新建配置项请求体（同名 group+key 会 upsert） */
export interface SettingCreateReq {
  // 配置分组（必填）
  group: string
  // 配置键名（必填）
  key: string
  // 值类型
  value_type?: string
  // 配置值（必填，明文）
  value: string
  // 是否加密落库
  encrypted?: boolean
  // 状态，默认 enabled
  status?: string
  // 描述
  description?: string
}

/** 更新配置项请求体（局部更新：空值字段不更新） */
export interface SettingUpdateReq {
  // 新值
  value?: string
  // 是否加密落库（配合 value）
  encrypted?: boolean
  // 状态
  status?: string
  // 描述
  description?: string
}

// ============================ 定时任务 Cron ============================
// golib gincron 路由，响应字段为 snake_case。

/** 定时任务 */
export interface CronTask {
  // 任务 ID
  id: string
  // 业务 ID
  biz_id: string
  // 业务类型
  biz_type: string
  // 任务名称
  name: string
  // 任务类型
  task_type: string
  // cron 表达式
  spec: string
  // 描述
  description: string
  // 状态：enabled / disabled
  status: string
  // 上次运行时间（unix 秒）
  last_run_at?: number | null
  // 下次运行时间（unix 秒）
  next_run_at?: number | null
  // 创建时间（unix 秒）
  created_at: number
  // 更新时间（unix 秒）
  updated_at: number
}

/** 定时任务运行记录 */
export interface CronTaskRun {
  // 运行记录 ID
  id: string
  // 所属任务 ID
  task_id: string
  // 开始时间（unix 秒）
  start_at: number
  // 结束时间（unix 秒）
  end_at?: number | null
  // 耗时（毫秒）
  duration_ms: number
  // 状态
  status: string
  // 错误信息
  error_msg: string
  // 请求链路 ID
  request_id: string
  // 创建时间（unix 秒）
  created_at: number
}

/** 定时任务分页结果 */
export interface CronTaskListResult {
  // 当前页任务
  list: CronTask[]
  // 总条数
  total: number
}

/** 定时任务运行记录分页结果 */
export interface CronTaskRunListResult {
  // 当前页运行记录
  list: CronTaskRun[]
  // 总条数
  total: number
}

// ============================ 异步任务 Async ============================
// golib ginasync 路由，响应字段为 snake_case。

/** 异步任务 */
export interface AsyncTask {
  // 任务 ID
  id: string
  // 任务名称
  name: string
  // 描述
  description: string
  // 状态：enabled / disabled
  status: string
  // 创建时间（unix 秒）
  created_at: number
  // 更新时间（unix 秒）
  updated_at: number
}

/** 异步任务运行记录 */
export interface AsyncTaskRun {
  // 运行记录 ID
  id: string
  // 任务类型
  task_type: string
  // 队列
  queue: string
  // 状态
  status: string
  // 已重试次数
  retried: number
  // 最大重试次数
  max_retry: number
  // 开始时间（unix 秒）
  start_at?: number | null
  // 结束时间（unix 秒）
  end_at?: number | null
  // 耗时（毫秒）
  duration_ms: number
  // 错误信息
  error_msg: string
  // 请求链路 ID
  request_id: string
  // 创建时间（unix 秒）
  created_at: number
}

/** 异步任务分页结果 */
export interface AsyncTaskListResult {
  // 当前页任务
  list: AsyncTask[]
  // 总条数
  total: number
}

/** 异步任务运行记录分页结果 */
export interface AsyncTaskRunListResult {
  // 当前页运行记录
  list: AsyncTaskRun[]
  // 总条数
  total: number
}

// ============================ 文件上传 Files ============================
// golib ginupload 路由，接口路径 /v1/demo/files，响应字段为 snake_case。

/** 文件记录（上传成功后返回，对应 ginupload.fileRecordResponse） */
export interface FileRecord {
  // 文件 ID
  file_id: string
  // 文件名
  name: string
  // MIME 类型
  mime_type: string
  // 文件状态：pending/uploading/completed/failed/aborted
  status: string
}

/** 下载地址响应（对应 ginupload.presignURLResponse） */
export interface PresignURLResp {
  // 预签名 URL
  url: string
  // 预签名请求的 HTTP 方法
  method: string
  // 签名覆盖的请求头，必须原样发送
  headers: Record<string, string[]>
  // 过期时间（秒）
  expires_in: number
}

// ============================ 健康检查 Health ============================

/** 依赖连通性检查结果：连通为 ok，否则为错误信息 */
export interface DbCheck {
  // PostgreSQL 连通性
  postgres: string
  // Redis 连通性
  redis: string
}
