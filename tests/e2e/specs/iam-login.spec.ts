import { test } from '../fixtures/base'

/**
 * 内嵌 IAM（原形态 op）浏览器级端到端——**已随 op 形态下线**。
 *
 * 原用例依赖本进程内嵌 OP 的产物（`/iam/login` 内嵌登录页、`/iam/config.json`、`/iam/install`
 * 初始化管理员），op 已下线，本模板不再内嵌 IAM 服务端。rp 形态（对接外部 ark-iam，OIDC 本地验签）
 * 需要**外部环境**（外部 OP + 已注册 client + 测试账号），无法在仓库内自举。
 *
 * TODO：未来搭建外部 rp 测试环境后恢复本用例——让后端以 `iam.mode: rp` 对接外部 OP、
 * 前端配 `VITE_OIDC_ISSUER` / `VITE_OIDC_CLIENT_ID`，再移除下方 `test.skip`。
 */
test.skip('内嵌 IAM 登录（浏览器级，op 已下线，rp 需外部环境）—— TODO 待外部 rp 环境恢复', async () => {
  // 占位：见文件头 TODO。恢复时补回内嵌登录流程与受保护接口断言。
})
