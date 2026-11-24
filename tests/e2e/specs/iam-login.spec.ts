import { test, expect } from '../fixtures/base'

/**
 * 内嵌 IAM（原形态 op）浏览器级端到端——**已随 op 形态下线整体跳过**。
 *
 * 该用例依赖本进程内嵌 OP 的产物：`/iam/login` 内嵌登录页、`/iam/config.json`、`/iam/install`
 * 初始化管理员。op 已下线，本模板不再内嵌 IAM 服务端；rp 形态（对接外部 ark-iam，OIDC 本地验签）
 * 需要**外部环境**（外部 OP + 已注册的 client + 测试账号），无法在仓库内自举，故整组 `describe.skip`。
 *
 * 保留文件（而非删除）以便未来搭建外部 rp 测试环境时按需恢复；如需恢复：移除 `.skip`，并让后端以
 * `iam.mode: rp` 对接外部 OP、前端配 `VITE_OIDC_ISSUER` / `VITE_OIDC_CLIENT_ID`。
 *
 * 原前置（已失效）：
 *   1. 后端 enable=true 启动，且 `/iam/install/initialize` 已初始化管理员；
 *   2. OIDC 客户端 redirect_uris 含 http://localhost:3000/auth/callback。
 */
test.describe.skip('内嵌 IAM 登录（浏览器级）——op 已下线，rp 需外部环境', () => {
  // 保留原凭据常量，便于未来恢复外部 rp 环境时直接使用（当前整组 skip，不会实际执行）。
  const ADMIN_USER = process.env.E2E_IAM_ADMIN_USER ?? 'admin'
  const ADMIN_PASS = process.env.E2E_IAM_ADMIN_PASS ?? 'Admin123456'

  test('未登录访问 → 跳内嵌登录页 → admin 登录 → 回跳业务页并拿到数据', async ({ page, request }) => {
    // 1. 未登录访问首页：AuthGuard 触发 signinRedirect，最终落到内嵌登录页 /iam/login。
    await page.goto('/')
    await expect(page).toHaveURL(/\/iam\/login\?authRequestID=/, { timeout: 20_000 })

    // 2. 内嵌登录页可见（antd 表单 + 登录按钮）。
    await expect(page.getByPlaceholder('请输入用户名')).toBeVisible({ timeout: 10_000 })
    await page.getByPlaceholder('请输入用户名').fill(ADMIN_USER)
    await page.getByPlaceholder('请输入密码').fill(ADMIN_PASS)
    // antd 会在两个汉字的按钮文案中间插入空格（渲染为「登 录」），故用正则匹配。
    await page.getByRole('button', { name: /登\s*录/ }).click()

    // 3. 登录成功回跳 demo 应用（origin localhost:3000，路径回到业务页，不再是 /iam/login）。
    await expect(page).toHaveURL(/^http:\/\/localhost:3000\//, { timeout: 20_000 })
    await expect(page).not.toHaveURL(/\/iam\/login/)

    // 4. 业务页渲染（商品管理页），证明带 token 已能访问受保护业务接口。
    await expect(page.getByRole('menuitem', { name: '商品管理' })).toBeVisible({ timeout: 15_000 })

    // 5. 强断言：登录后取到的 access_token 能打通受保护接口，且审计列=映射 uid（非 0）。
    //    直接用页面上下文 fetch（同源，带 AuthGuard 注入的 token），验证 200 + code 0。
    const token = await page.evaluate(async () => {
      // 门面包把 token 存在 sessionStorage（react-oidc-context 默认 userStore）。
      for (let i = 0; i < sessionStorage.length; i++) {
        const k = sessionStorage.key(i)!
        const v = sessionStorage.getItem(k) || ''
        const m = v.match(/"access_token":"([^"]+)"/)
        if (m) return m[1]
      }
      return null
    })
    expect(token, 'sessionStorage 中应存在 access_token').toBeTruthy()

    const resp = await request.get(
      `${process.env.E2E_API_BASE ?? 'http://localhost:8099'}/v1/demo/products?page=1&pageSize=2`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body.code).toBe(0)
  })

  test('无 token 直接打受保护接口 → 401（fail-closed）', async ({ request }) => {
    const resp = await request.get(
      `${process.env.E2E_API_BASE ?? 'http://localhost:8099'}/v1/demo/products?page=1&pageSize=2`,
    )
    expect(resp.status()).toBe(401)
  })
})
