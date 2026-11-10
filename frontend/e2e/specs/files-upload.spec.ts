import { createHash } from 'node:crypto'
import { test, expect } from '../fixtures/base'
import { deleteFile, checkFileExists, serveFile } from '../support/api'

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
})
