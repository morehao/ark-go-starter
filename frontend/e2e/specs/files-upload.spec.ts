import { createHash, randomBytes } from 'node:crypto'
import { test, expect } from '../fixtures/base'
import { API_BASE, deleteFile, checkFileExists, serveFile } from '../support/api'

/**
 * 文件上传页 · 全流程（上传 → 展示文件信息 → 下载 → 删除，含重名/秒传探测）。
 *
 * 回归重点：后端 POST /v1/demo/files 的 content_hash 是必填（ginupload.UploadRequest 上
 * binding:"required"，且 64 位 hex 会与内容校验），前端必须先算 SHA-256 再上传，
 * 否则稳定收到 {"code":-1,"msg":"content_hash is required"}。本用例是该回归的端到端护栏。
 *
 * 断言定位说明（避免 strict mode 误伤）：
 * - 「文件信息」既是 Descriptions 标题、也出现在 PageContainer description 里 → 用 exact: true；
 * - file_id 直接从上传响应体取，不靠肉眼文本正则；
 * - Popconfirm 的「确定」按钮用 .ant-popconfirm 限定作用域；antd 两字按钮会插空格 → 用 /删\s*除/；
 * - 下载断言分两层：UI 点击「下载」触发浏览器 download 事件（文件名正确）+ 直接请求 /serve 校验内容字节一致。
 *
 * 数据：文件名带 e2e- 前缀 + 时间戳；用例内 UI 删除，afterEach 走接口按 id 兜底清理。
 */
const FILE_PREFIX = 'e2e-upload'

interface UploadResp {
  code: number
  msg: string
  data?: { file_id?: string; name?: string; mime_type?: string; status?: string }
}

/** 计算内容 sha256（与页面 sha256Hex 等价），用于 check-exist 断言。 */
function sha256Hex(content: string): string {
  return createHash('sha256').update(content).digest('hex')
}

/** 计算二进制内容的 sha256，用于与后端记录的 content_hash / 直出内容对账。 */
function sha256Buffer(buffer: Buffer): string {
  return createHash('sha256').update(buffer).digest('hex')
}

/** 与页面阈值一致：> 8 MiB 走分片上传；9 MiB 文件配合 8 MiB 分片正好两片。 */
const BIG_FILE_BYTES = 9 * 1024 * 1024

/** 判断某个响应是否为「第 N 片」的预签名请求（按请求体 part_number 区分）。 */
function isPresignPartResponse(
  response: import('@playwright/test').Response,
  partNumber: number,
): boolean {
  const path = new URL(response.url()).pathname
  if (response.request().method() !== 'POST' || !path.endsWith('/parts')) return false
  const postData = response.request().postData()
  if (!postData) return false
  try {
    return (JSON.parse(postData) as { part_number?: number }).part_number === partNumber
  } catch {
    return false
  }
}

/** 进入文件上传页。 */
async function gotoFiles(page: import('@playwright/test').Page) {
  await page.goto('/')
  await page.getByRole('menuitem', { name: '文件上传' }).click()
  await expect(page.getByRole('heading', { name: '文件上传', level: 5 })).toBeVisible()
}

/** 通过 antd Upload 的隐藏 input 选文件，返回上传接口响应。 */
async function uploadViaUI(
  page: import('@playwright/test').Page,
  fileName: string,
  content: string,
): Promise<UploadResp> {
  const [uploadResp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/v1/demo/files') && r.request().method() === 'POST',
    ),
    page.locator('input[type="file"]').setInputFiles({
      name: fileName,
      mimeType: 'text/plain',
      buffer: Buffer.from(content),
    }),
  ])
  const body = (await uploadResp.json()) as UploadResp
  expect(body.code, `上传接口返回业务错误：${body.msg}`).toBe(0)
  await expect(page.getByText('上传成功')).toBeVisible()
  return body
}

test.describe('文件上传 · 全流程', () => {
  const uploadedFileIDs: string[] = []

  test.afterEach(async () => {
    // 兜底清理：用例中途失败时 UI 删除可能没走到，这里用接口把残留文件删掉。
    for (const id of uploadedFileIDs.splice(0)) {
      await deleteFile(id).catch(() => undefined)
    }
  })

  test('上传 → 展示文件信息 → 下载内容一致 → 删除', async ({ page, request }) => {
    const fileName = `${FILE_PREFIX}-${Date.now()}.txt`
    const content = `e2e upload payload ${fileName}\nline2\n`

    await gotoFiles(page)
    const body = await uploadViaUI(page, fileName, content)
    const fileID = body.data?.file_id
    expect(fileID, '上传响应必须带 file_id').toBeTruthy()
    uploadedFileIDs.push(fileID!)

    // 文件信息面板：标题 exact、文件名回显、状态 Tag「已完成」
    await expect(page.getByText('文件信息', { exact: true })).toBeVisible()
    await expect(page.getByText(fileName, { exact: true })).toBeVisible()
    await expect(page.getByText('已完成', { exact: true })).toBeVisible()
    // 详情拉取完成后下载按钮可见（file_id → GET 详情链路通）
    await expect(page.getByRole('button', { name: /下\s*载/ })).toBeVisible()

    // 下载：UI 点击触发浏览器 download 事件，文件名应与上传文件名一致
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: /下\s*载/ }).click(),
    ])
    expect(download.suggestedFilename()).toBe(fileName)
    expect(download.url()).toContain(`/v1/demo/files/${fileID}/serve`)

    // /serve 直出内容必须与上传内容逐字节一致（端到端内容正确性）
    const served = await serveFile(fileID!)
    expect(served.status).toBe(200)
    expect(served.contentType).toContain('text/plain')
    expect(served.text).toBe(content)

    // 删除：Popconfirm 二次确认
    await page.getByRole('button', { name: /删\s*除/ }).click()
    await page
      .locator('.ant-popconfirm')
      .getByRole('button', { name: /确\s*定|OK/i })
      .click()
    await expect(page.getByText('删除成功')).toBeVisible()
    await expect(page.getByText(fileName, { exact: true })).toHaveCount(0)

    // 删除语义复核：详情接口返回业务错误、serve 不再返回文件内容
    const detailResp = await request.get(`/v1/demo/files/${fileID}`)
    expect(detailResp.status()).toBe(200)
    expect(((await detailResp.json()) as { code: number }).code).not.toBe(0)
    const servedAfterDelete = await serveFile(fileID!)
    expect(servedAfterDelete.status).toBe(200)
    expect(servedAfterDelete.text).not.toBe(content)

    // 已删除则无需兜底
    uploadedFileIDs.length = 0
  })

  test('同名不同内容二次上传：两条记录各自可下载，内容不串', async ({ page }) => {
    const stamp = Date.now()
    const fileName = `${FILE_PREFIX}-same-name-${stamp}.txt`
    const firstContent = `first content ${stamp}\n`
    const secondContent = `second content ${stamp}\n`

    await gotoFiles(page)
    const first = await uploadViaUI(page, fileName, firstContent)
    const firstID = first.data?.file_id
    expect(firstID).toBeTruthy()
    uploadedFileIDs.push(firstID!)

    // 同名不同内容仍应上传成功（不做文件名唯一约束），且拿到新的 file_id
    const second = await uploadViaUI(page, fileName, secondContent)
    const secondID = second.data?.file_id
    expect(secondID, '第二次上传应返回 file_id').toBeTruthy()
    uploadedFileIDs.push(secondID!)

    // 面板展示的是最新一次上传的文件名
    await expect(page.getByText(fileName, { exact: true })).toBeVisible()

    // 两次内容各自可下载、互不覆盖
    const servedFirst = await serveFile(firstID!)
    const servedSecond = await serveFile(secondID!)
    expect(servedFirst.text).toBe(firstContent)
    expect(servedSecond.text).toBe(secondContent)
  })

  test('秒传探测：check-exist 命中已上传内容哈希', async ({ page }) => {
    const fileName = `${FILE_PREFIX}-exist-${Date.now()}.txt`
    const content = `check-exist payload ${Date.now()}\n`
    const hash = sha256Hex(content)

    await gotoFiles(page)
    // 上传前：该内容尚不存在，且不返回文件记录
    const before = await checkFileExists(hash)
    expect(before.exists).toBe(false)
    expect(before.file).toBeFalsy()

    const body = await uploadViaUI(page, fileName, content)
    const fileID = body.data?.file_id
    expect(fileID).toBeTruthy()
    uploadedFileIDs.push(fileID!)

    // 上传后：同一内容哈希命中去重索引，且返回可用的文件记录
    // （golib v1.34.0 曾返回空 file 对象，v1.34.1 修复为完整 file_id/name/mime/status）
    const after = await checkFileExists(hash)
    expect(after.exists).toBe(true)
    expect(after.file, 'exists=true 时必须返回文件记录').toBeTruthy()
    expect(after.file?.file_id, '秒传记录必须带可用 file_id').toBeTruthy()
    expect(after.file?.name).toBe(fileName)
    expect(after.file?.mime_type).toBe('text/plain')
    expect(after.file?.status).toBe('completed')
    // 秒传拿到的 id 必须真的能下载同一份内容
    const served = await serveFile(after.file!.file_id)
    expect(served.status).toBe(200)
    expect(served.text).toBe(content)
  })

  test('超大文件（9 MB）走分片上传：create → parts → 分片直传 → complete，内容与 content_hash 一致', async ({
    page,
    request,
  }) => {
    await gotoFiles(page)

    const fileName = `${FILE_PREFIX}-big-${Date.now()}.bin`
    // 随机内容保证跨轮次唯一：同 content_hash 会命中后端去重（create 返回 409）
    const content = randomBytes(BIG_FILE_BYTES)
    const contentHash = sha256Buffer(content)

    // 记录所有分片 PUT 响应：既断言数量，也断言直传走的是同源地址（跨源预签名 host 会被 CORS 拦）
    const putResponses: import('@playwright/test').Response[] = []
    page.on('response', (response) => {
      if (response.request().method() !== 'PUT') return
      if (!new URL(response.url()).pathname.includes('/v1/demo/objects/')) return
      putResponses.push(response)
    })

    // 三个业务接口 + 两片预签名：先挂好等待，再触发上传，避免漏掉事件
    const createRespPromise = page.waitForResponse((r) => {
      const path = new URL(r.url()).pathname
      return r.request().method() === 'POST' && path.endsWith('/v1/demo/files/multipart')
    })
    const part1RespPromise = page.waitForResponse((r) => isPresignPartResponse(r, 1))
    const part2RespPromise = page.waitForResponse((r) => isPresignPartResponse(r, 2))
    const completeRespPromise = page.waitForResponse((r) => {
      const path = new URL(r.url()).pathname
      return (
        r.request().method() === 'POST' &&
        /\/v1\/demo\/files\/multipart\/[^/]+\/complete$/.test(path)
      )
    })

    await page.locator('input[type="file"]').setInputFiles({
      name: fileName,
      mimeType: 'application/octet-stream',
      buffer: content,
    })

    // ① 创建会话
    const createBody = (await (await createRespPromise).json()) as {
      code: number
      msg: string
      data?: { file_id?: string; upload_id?: string }
    }
    expect(createBody.code, `创建分片会话失败：${createBody.msg}`).toBe(0)
    const fileID = createBody.data?.file_id
    expect(fileID, '创建会话必须返回 file_id').toBeTruthy()
    uploadedFileIDs.push(fileID!)

    // ② 两片预签名：都返回可直传的 PUT 地址
    for (const [partNumber, respPromise] of [
      [1, part1RespPromise],
      [2, part2RespPromise],
    ] as const) {
      const body = (await (await respPromise).json()) as {
        code: number
        msg: string
        data?: { url?: string; method?: string }
      }
      expect(body.code, `第 ${partNumber} 片预签名失败：${body.msg}`).toBe(0)
      expect(body.data?.url, `第 ${partNumber} 片必须返回预签名 URL`).toBeTruthy()
      expect(body.data?.method).toBe('PUT')
    }

    // ③ 分片直传：9 MiB + 8 MiB 分片 = 两片，且必须经同源 /v1 代理（不能直连 8099）
    await expect.poll(() => putResponses.length, { timeout: 15_000 }).toBe(2)
    const origin = new URL(page.url()).origin
    for (const response of putResponses) {
      expect(response.status()).toBe(200)
      expect(
        new URL(response.url()).origin,
        '分片直传必须走同源 /v1 代理（跨源会被 CORS 预检拦截）',
      ).toBe(origin)
    }

    // ④ complete：合并后返回的记录必须与创建会话是同一个 file_id
    const completeBody = (await (await completeRespPromise).json()) as UploadResp
    expect(completeBody.code, `合并分片失败：${completeBody.msg}`).toBe(0)
    expect(completeBody.data?.file_id).toBe(fileID)
    expect(completeBody.data?.status).toBe('completed')
    expect(completeBody.data?.name).toBe(fileName)

    // UI：成功提示 + 文件信息面板
    await expect(page.getByText(/上传成功/)).toBeVisible()
    await expect(page.getByText('文件信息', { exact: true })).toBeVisible()
    await expect(page.getByText(fileName, { exact: true })).toBeVisible()
    await expect(page.getByText('已完成', { exact: true })).toBeVisible()

    // 后端记录的 content_hash / size 必须与客户端增量摘要一致（增量哈希正确性护栏）
    const detailResp = await request.get(`${API_BASE}/v1/demo/files/${fileID}`)
    expect(detailResp.status()).toBe(200)
    const detail = (await detailResp.json()) as {
      code: number
      data?: { content_hash?: string; size?: number; status?: string }
    }
    expect(detail.code).toBe(0)
    expect(detail.data?.status).toBe('completed')
    expect(detail.data?.size).toBe(BIG_FILE_BYTES)
    expect(detail.data?.content_hash, 'content_hash 必须等于整个文件的 SHA-256').toBe(contentHash)

    // 合并后的对象字节必须与上传内容逐字节一致（分片顺序/拼接正确性）
    const served = await request.get(`${API_BASE}/v1/demo/files/${fileID}/serve`)
    expect(served.status()).toBe(200)
    const servedBytes = await served.body()
    expect(servedBytes.length).toBe(BIG_FILE_BYTES)
    expect(sha256Buffer(servedBytes)).toBe(contentHash)
  })

  test('分片直传失败：清理已创建的分片会话（abort）并提示错误', async ({ page }) => {
    await gotoFiles(page)

    const fileName = `${FILE_PREFIX}-abort-${Date.now()}.bin`
    // 只拦截对象直传端点的 PUT（模拟网络失败）；创建/预签名/abort 等业务接口不受影响
    await page.route('**/v1/demo/objects/**', (route) => route.abort('failed'))

    const createRespPromise = page.waitForResponse((r) => {
      const path = new URL(r.url()).pathname
      return r.request().method() === 'POST' && path.endsWith('/v1/demo/files/multipart')
    })
    const abortRespPromise = page.waitForResponse((r) => {
      const path = new URL(r.url()).pathname
      return r.request().method() === 'DELETE' && /\/v1\/demo\/files\/multipart\/[^/]+$/.test(path)
    })

    await page.locator('input[type="file"]').setInputFiles({
      name: fileName,
      mimeType: 'application/octet-stream',
      buffer: randomBytes(BIG_FILE_BYTES),
    })

    // 直传失败前会话已创建，失败后必须以 DELETE /files/multipart/{id} 清理（不留下半截分片）
    const createBody = (await (await createRespPromise).json()) as {
      code: number
      data?: { file_id?: string }
    }
    const fileID = createBody.data?.file_id
    expect(fileID, '直传失败前应已创建分片会话').toBeTruthy()
    uploadedFileIDs.push(fileID!)

    await expect(page.locator('.ant-message-error')).toBeVisible()
    const abortResp = await abortRespPromise
    expect(abortResp.status()).toBe(200)
    expect(((await abortResp.json()) as { code: number }).code).toBe(0)
    // 失败不应展示文件信息面板
    await expect(page.getByText('文件信息', { exact: true })).toHaveCount(0)
  })
})
