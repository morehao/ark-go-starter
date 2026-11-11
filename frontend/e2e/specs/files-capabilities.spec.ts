import { createHash, randomBytes } from 'node:crypto'
import { test, expect } from '../fixtures/base'
import {
  API_BASE,
  deleteFile,
  getFileDetail,
  listFileParts,
  presignFile,
  redirectFileByQuery,
  serveFile,
  serveFileByQuery,
  uploadFileDirect,
} from '../support/api'

/**
 * 文件服务 · 完整路由能力（对应页面 ② 详情 / ③ 四种取回方式 / ④ 分片调试）。
 *
 * 与 files-upload.spec.ts（上传主流程）互补：这里聚焦「页面已展示、但原 e2e 未覆盖」的
 * ginupload 端点契约 —— 文件详情全量字段、presign-url、redirect(302)、serve(query)、
 * redirect(query)、parts 列举 —— 直接走 REST 校验后端契约，并做一条 UI 冒烟
 * （presign-url 按钮回显可用地址），避免强依赖具体按钮文案。
 *
 * 造数走 REST（uploadFileDirect）而不是 UI，聚焦能力本身；每个用例 afterEach 按 id 删除。
 */

const FILE_PREFIX = 'e2e-caps'

/** 计算文本内容的 sha256（小写 hex），与后端 content_hash 语义一致。 */
function sha256Hex(content: string): string {
  return createHash('sha256').update(content).digest('hex')
}

/** 进入文件上传页（菜单项文案固定为「文件上传」）。 */
async function gotoFiles(page: import('@playwright/test').Page) {
  await page.goto('/')
  await page.getByRole('menuitem', { name: '文件上传' }).click()
  await expect(page.getByRole('heading', { name: '文件上传', level: 5 })).toBeVisible()
}

test.describe('文件服务 · 完整路由能力', () => {
  const createdIDs: string[] = []

  test.afterEach(async () => {
    for (const id of createdIDs.splice(0)) {
      await deleteFile(id).catch(() => undefined)
    }
  })

  test('详情：GET /files/{id} 返回全量字段且与上传内容一致', async () => {
    const name = `${FILE_PREFIX}-detail-${Date.now()}.txt`
    const content = `detail payload ${name}\n`
    const hash = sha256Hex(content)
    const rec = await uploadFileDirect(name, Buffer.from(content), hash)
    createdIDs.push(rec.file_id)

    const detail = await getFileDetail(rec.file_id)
    expect(detail.file_id).toBe(rec.file_id)
    expect(detail.content_hash).toBe(hash)
    expect(detail.name).toBe(name)
    expect(detail.size).toBe(Buffer.byteLength(content))
    // 详见：uploadFileDirect 用 Blob 直传且未显式设置 type，服务端按分片头推导为 octet-stream；
    // 这里只断言 mime_type 字段存在且非空（字段回传正确性），不耦合具体值。
    expect(detail.mime_type, 'mime_type 字段应存在').toBeTruthy()
    expect(detail.storage_uri, 'local 驱动应返回 storage_uri').toBeTruthy()
    expect(detail.status).toBe('completed')
    expect(detail.created_at).toBeTruthy()
    expect(detail.updated_at).toBeTruthy()
  })

  test('presign-url：返回的地址可安全获取文件内容', async () => {
    const name = `${FILE_PREFIX}-presign-${Date.now()}.txt`
    const content = `presign payload ${name}\n`
    const rec = await uploadFileDirect(name, Buffer.from(content), sha256Hex(content))
    createdIDs.push(rec.file_id)

    const presigned = await presignFile(rec.file_id)
    expect(presigned.url, 'presign-url 必须返回 url').toBeTruthy()
    // local 驱动的预签名 url 指向本服务的 /v1/demo/objects/...（含 token/expires）
    expect(new URL(presigned.url).pathname).toContain('/v1/demo/objects/')
    expect(presigned.expires_in).toBeGreaterThan(0)

    // 预签名 GET 应能读回原文内容（token 覆盖读操作）
    const got = await fetch(presigned.url)
    expect(got.status).toBe(200)
    expect(await got.text()).toBe(content)
  })

  test('redirect：路径与 query 两种形式都 302 到可读文件 URL', async () => {
    const name = `${FILE_PREFIX}-redirect-${Date.now()}.txt`
    const content = `redirect payload ${name}\n`
    const rec = await uploadFileDirect(name, Buffer.from(content), sha256Hex(content))
    createdIDs.push(rec.file_id)

    // 路径形式 GET /files/{id}/redirect → 302（fetch 手动重定向以读取 Location）
    const pathRedirect = await fetch(`${API_BASE}/v1/demo/files/${rec.file_id}/redirect`, {
      redirect: 'manual',
    })
    expect(pathRedirect.status).toBe(302)
    const loc = pathRedirect.headers.get('location') ?? ''
    expect(loc).toBeTruthy()

    // query 形式 GET /files/redirect?file_id= → 302
    const queryRedirect = await redirectFileByQuery({ fileId: rec.file_id })
    expect(queryRedirect.status).toBe(302)
    expect(queryRedirect.location).toBeTruthy()

    // 跟随 query 形式 Location 能读到原文
    const followed = await fetch(queryRedirect.location)
    expect(followed.status).toBe(200)
    expect(await followed.text()).toBe(content)
  })

  test('serve(query)：?file_id= 与 ?storage_uri= 都能直出原文', async () => {
    const name = `${FILE_PREFIX}-serveq-${Date.now()}.txt`
    const content = `serve-query payload ${name}\n`
    const rec = await uploadFileDirect(name, Buffer.from(content), sha256Hex(content))
    createdIDs.push(rec.file_id)

    const byId = await serveFileByQuery({ fileId: rec.file_id })
    expect(byId.status).toBe(200)
    expect(byId.text).toBe(content)

    const detail = await getFileDetail(rec.file_id)
    const byURI = await serveFileByQuery({ storageUri: detail.storage_uri })
    expect(byURI.status).toBe(200)
    expect(byURI.text).toBe(content)

    // 对照：路径形式 /serve 内容一致
    const byPath = await serveFile(rec.file_id)
    expect(byPath.text).toBe(content)
  })

  test('parts：分片上传后的文件可列举到分片；普通上传文件无分片', async () => {
    // 构造一个分片会话：create → 逐片 presign + PUT → complete，得到一个完成的分片文件记录
    const bigName = `${FILE_PREFIX}-big-${Date.now()}.bin`
    const bigContent = randomBytes(9 * 1024 * 1024)
    const bigHash = createHash('sha256').update(bigContent).digest('hex')

    const createResp = await fetch(`${API_BASE}/v1/demo/files/multipart`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content_hash: bigHash,
        name: bigName,
        size: bigContent.length,
        mime_type: 'application/octet-stream',
      }),
    })
    const createBody = (await createResp.json()) as {
      code: number
      msg: string
      data?: { file_id?: string; upload_id?: string }
    }
    expect(createBody.code, `创建分片会话失败：${createBody.msg}`).toBe(0)
    const fileID = createBody.data?.file_id
    expect(fileID, '创建会话必须返回 file_id').toBeTruthy()
    createdIDs.push(fileID!)

    // 8 MiB 分片：9 MiB 内容 → 两片
    const PART_SIZE = 8 * 1024 * 1024
    const partsTotal = Math.ceil(bigContent.length / PART_SIZE)
    const parts: Array<{ part_number: number; etag: string }> = []
    for (let i = 0; i < partsTotal; i += 1) {
      const partNumber = i + 1
      const presignResp = await fetch(`${API_BASE}/v1/demo/files/multipart/${fileID}/parts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ part_number: partNumber }),
      })
      const presignBody = (await presignResp.json()) as {
        code: number
        data?: { url?: string; method?: string; headers?: Record<string, string[]> }
      }
      expect(presignBody.code).toBe(0)
      const slice = bigContent.subarray(i * PART_SIZE, Math.min((i + 1) * PART_SIZE, bigContent.length))
      const putHeaders = new Headers()
      for (const [k, vals] of Object.entries(presignBody.data!.headers ?? {})) {
        for (const v of Array.isArray(vals) ? vals : [String(vals)]) putHeaders.append(k, v)
      }
      const put = await fetch(presignBody.data!.url!, {
        method: presignBody.data!.method || 'PUT',
        headers: putHeaders,
        body: slice,
      })
      expect(put.status).toBe(200)
      const putBody = (await put.json()) as { data?: { etag?: string } }
      const etag = putBody.data?.etag ?? put.headers.get('etag')?.replace(/"/g, '')
      expect(etag, '分片 PUT 必须返回 etag').toBeTruthy()
      parts.push({ part_number: partNumber, etag: etag! })
    }

    // 关键：在 complete 之前列举已上传分片（会话仍活跃时才可列举；complete 后会话被清理）。
    // 这一步正是页面「④ 分片会话调试」调用的 GET /files/{id}/parts。
    const listedBeforeComplete = await listFileParts(fileID!, { maxParts: 100 })
    expect(listedBeforeComplete.parts.length, '会话活跃时应能列举到已上传分片').toBe(partsTotal)
    expect(listedBeforeComplete.parts.map((p) => p.part_number)).toEqual(
      parts.map((p) => p.part_number),
    )
    expect(listedBeforeComplete.parts.every((p) => p.etag)).toBe(true)
    expect(typeof listedBeforeComplete.is_truncated).toBe('boolean')

    const completeResp = await fetch(`${API_BASE}/v1/demo/files/multipart/${fileID}/complete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ parts }),
    })
    const completeBody = (await completeResp.json()) as { code: number; msg: string }
    expect(completeBody.code, `合并分片失败：${completeBody.msg}`).toBe(0)

    // complete 成功后会话被清理，此时列举分片应报错（not a multipart upload）——页面据此提示「无分片」
    const afterComplete = await listFileParts(fileID!).catch(() => null)
    expect(afterComplete, 'complete 会话清理后再列举分片应报错').toBeNull()

    // 对照：普通上传（非分片）的文件列举分片应报错（业务码非 0）或空列表，不崩溃
    const normalName = `${FILE_PREFIX}-normal-${Date.now()}.txt`
    const normalContent = `normal ${normalName}\n`
    const normalRec = await uploadFileDirect(
      normalName,
      Buffer.from(normalContent),
      sha256Hex(normalContent),
    )
    createdIDs.push(normalRec.file_id)
    const normalParts = await listFileParts(normalRec.file_id).catch(() => null)
    // 允许两种后端行为：报错（返回 null）或返回空 parts；都不应影响页面
    if (normalParts) {
      expect(normalParts.parts.length).toBe(0)
    }
  })

  test('UI 冒烟：上传后详情全字段展示 + presign-url 按钮回显可用地址 + 分片调试区可见', async ({
    page,
  }) => {
    const name = `${FILE_PREFIX}-ui-${Date.now()}.txt`
    const content = `ui smoke ${name}\n`
    await gotoFiles(page)

    // 经 UI 上传（隐藏 input），等待上传接口响应
    const [uploadResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/v1/demo/files') && r.request().method() === 'POST',
      ),
      page.locator('input[type="file"]').setInputFiles({
        name,
        mimeType: 'text/plain',
        buffer: Buffer.from(content),
      }),
    ])
    const uploadBody = (await uploadResp.json()) as {
      code: number
      data?: { file_id?: string }
    }
    expect(uploadBody.code).toBe(0)
    createdIDs.push(uploadBody.data!.file_id!)

    // ② 详情区：标题 + 文件名 + 内容哈希字段标签
    await expect(page.getByText('文件信息', { exact: true })).toBeVisible()
    await expect(page.getByText(name, { exact: true })).toBeVisible()
    await expect(page.getByText('内容哈希')).toBeVisible()

    // ③ 访问方式区出现 presign-url 按钮，点击后回显 URL（本页渲染「URL：」+ 可点「打开」）
    await expect(page.getByText('下载与访问方式', { exact: true })).toBeVisible()
    const presignBtn = page.getByRole('button', { name: '获取预签名下载地址' })
    await expect(presignBtn).toBeVisible()
    const [presignResp] = await Promise.all([
      page.waitForResponse(
        (r) => /\/v1\/demo\/files\/[^/]+\/presign-url$/.test(new URL(r.url()).pathname),
      ),
      presignBtn.click(),
    ])
    const presignBody = (await presignResp.json()) as { code: number; data?: { url?: string } }
    expect(presignBody.code).toBe(0)
    expect(presignBody.data?.url).toBeTruthy()
    // 回显区出现该 url 文本（截断也可能，用包含 pathname 的关键片段）
    await expect(page.getByText('/v1/demo/objects/', { exact: false })).toBeVisible()

    // ④ 分片调试区可见
    await expect(page.getByText('分片会话调试', { exact: true })).toBeVisible()
  })
})
