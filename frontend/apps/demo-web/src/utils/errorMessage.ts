/**
 * 后端/传输层错误文案 → 中文展示文案的集中映射（显示层本地化）。
 *
 * 背景：后端 golib 的 `gincontext.Fail` 会把 English error 原文直接放进信封的 `msg`
 * （如 `dict: type still has items`、`setting not found`），api 层把 msg 包成 `Error`
 * 抛出，页面若直接 `message.error(e.message)` 就会把英文原文暴露给用户。
 *
 * 本模块只影响**展示**，不改变后端契约、请求参数与错误语义：
 * - 已知 msg：精确保留地映射为中文；
 * - 传输层错误（axios 网络/超时/HTTP 状态码、参数绑定失败前缀）按模式映射；
 * - 未知 msg：原样返回，避免新错误信息被吞掉（便于排障）。
 *
 * 新增后端错误文案时：在 `MESSAGE_MAP` 里补一条即可，不要在各页面内散落判断。
 */

/** 后端 msg 精确映射表（key 为后端英文原文，value 为中文提示） */
const MESSAGE_MAP: Record<string, string> = {
  // ---- 数据字典 dict ----
  'dict: db is required': '字典服务未就绪：数据库未配置',
  'dict: type code is required': '类型编码不能为空',
  'dict: item value is required': '字典项码值不能为空',
  'dict: item label is required': '字典项展示文案不能为空',
  'dict: invalid type code format': '类型编码格式不合法',
  'dict: invalid item value format': '字典项码值格式不合法',
  'dict: extra is not valid json': '扩展字段不是合法的 JSON',
  'dict: invalid status, expect enabled or disabled': '状态取值不合法，仅支持启用/禁用',
  'dict: type code already exists': '类型编码已存在，请更换',
  'dict: item value already exists in this type': '该字典类型下码值已存在，请更换',
  'dict: dict type not found': '字典类型不存在或已被删除',
  'dict: dict type is disabled': '字典类型已禁用，无法操作',
  'dict: dict item not found': '字典项不存在或已被删除',
  'dict: dict item is disabled': '字典项已禁用，无法操作',
  'dict: item still has children': '该字典项下仍有子项，无法删除',
  'dict: type still has items': '该字典类型下仍有字典项，无法删除',
  'dict: cannot move item into itself or its descendant': '不能把字典项移动到自身或其子节点下',
  'dict: tree level exceeds max level': '树层级超过系统允许的最大层级',
  'dict: parent item belongs to another type': '父项属于其它字典类型，无法挂载',
  'dict: record not found': '记录不存在或已被删除',
  'dict: not initialized, call Init first': '字典服务未初始化',
  'dict: batch size exceeds limit': '批量条数超过上限',

  // ---- 配置中心 setting ----
  'setting not found': '配置项不存在或已被删除',
  'group and key are required': '分组和键名不能为空',
  'value is empty': '配置值不能为空',
  'invalid value type': '值类型不合法',
  'unsupported value type': '不支持该值类型',
  'invalid status, expect enabled or disabled': '状态取值不合法，仅支持启用/禁用',
  'value is required when encryption is enabled': '开启加密存储时必须提供配置值',
  'crypto key not configured': '未配置加密密钥，无法加密存储',
  'invalid ciphertext format': '密文格式不合法，无法解密',
  'no codec registered for value type': '该值类型未注册编解码器',
  'db is required': '服务未就绪：数据库未配置',
  'not initialized, call Init first': '配置服务未初始化',

  // ---- 定时任务 cron ----
  'gcron: task not registered': '定时任务未注册或已被移除',
  'gcron: run not found': '运行记录不存在',
  'gcron: duplicate task id': '定时任务 ID 重复',
  'gcron: task id is empty': '定时任务 ID 不能为空',
  'gcron: task type is empty': '定时任务类型不能为空',
  'gcron: task spec is empty': 'Cron 表达式不能为空',
  'gcron: db is nil': '定时任务存储未就绪',
  'gcron: distlock factory not configured': '定时任务分布式锁未配置',

  // ---- 异步任务 async ----
  'gasync: task not found': '异步任务类型未注册',
  'gasync: run not found': '运行记录不存在',
  'gasync: duplicate task type': '异步任务类型重复注册',
  'gasync: task type name is empty': '异步任务类型名不能为空',
  'gasync: handler is nil': '异步任务处理器为空',
  'gasync: db is nil': '异步任务存储未就绪',
  'gasync: redis addr is empty': '异步任务 Redis 地址未配置',
}

/** 传输层错误文案（axios 内置 message） */
const NETWORK_ERROR = 'Network Error'

/** 把后端/传输层原文转换为中文展示文案；未知原文原样返回 */
export function localizeMessage(message: string): string {
  const msg = message.trim()
  if (!msg) return ''

  const known = MESSAGE_MAP[msg]
  if (known) return known

  if (msg === NETWORK_ERROR) return '网络异常，请检查后端服务是否可用'

  // axios 超时：`timeout of 10000ms exceeded`
  if (/^timeout of \d+ms exceeded$/i.test(msg)) return '请求超时，请稍后重试'

  // axios HTTP 层非 2xx：`Request failed with status code 500`
  const httpStatus = /^Request failed with status code (\d{3})$/i.exec(msg)
  if (httpStatus) return `请求失败（HTTP ${httpStatus[1]}），请稍后重试`

  // 后端参数绑定失败统一以 `invalid request: <Go binding detail>` 返回，detail 是英文内部信息，不外露
  if (/^invalid request:\s*/i.test(msg)) return '请求参数不合法，请检查填写内容后重试'

  return msg
}

/**
 * 页面统一错误提示取文案入口：
 * - `Error` → 用其 message 做本地化；空 message 时回落到页面兜底文案；
 * - 字符串 → 按原文本地化；
 * - 其它 → 页面兜底文案。
 */
export function localizeError(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    return localizeMessage(error.message) || fallback
  }
  if (typeof error === 'string') {
    return localizeMessage(error) || fallback
  }
  return fallback
}
