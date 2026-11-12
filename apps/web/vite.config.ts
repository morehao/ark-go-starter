import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

/**
 * 第三方依赖拆包规则（按顺序匹配，命中第一条即停止）。
 *
 * 背景：demo-web 未引入路由库的代码分割，所有页面同步引入，依赖全部打进
 * 单个 index chunk，未压缩体积超过 1.2 MB 触发 Vite 的 500 kB 告警。
 * 这里按「稳定但不细碎」的粒度拆成 vendor chunk，保持长期缓存友好，
 * 单个 chunk 均远低于 500 kB（未压缩），构建无体积告警。
 *
 * 规则只用包名匹配，不依赖具体业务模块路径，新增页面无需改这里。
 */
const VENDOR_CHUNKS: ReadonlyArray<{ name: string; test: RegExp }> = [
  // React 运行时核心：react/react-dom/scheduler/react-is
  { name: 'react-vendor', test: /[\\/]node_modules[\\/](react|react-dom|scheduler|react-is)[\\/]/ },
  // antd 本体的组件实现（体积最大，单独成 chunk 便于长期缓存）
  { name: 'antd-vendor', test: /[\\/]node_modules[\\/]antd[\\/]/ },
  // antd 生态与其底层无样式组件（@ant-design/* + @rc-component/*）合成一个 chunk：
  // 两者存在双向引用，拆成两个 chunk 会触发 rollup 的 Circular chunk 警告。
  {
    name: 'antd-deps',
    test: /[\\/]node_modules[\\/](@ant-design[\\/]|@rc-component[\\/]|rc-[^\\/]+[\\/])/,
  },
  // IAM / 路由相关：@morehao/ark-iam-react、react-oidc-context、oidc-client-ts、react-router*
  {
    name: 'iam-vendor',
    test: /[\\/]node_modules[\\/](@morehao[\\/]ark-iam-react|react-oidc-context|oidc-client-ts|react-router|react-router-dom)[\\/]/,
  },
  // HTTP 客户端
  { name: 'http-vendor', test: /[\\/]node_modules[\\/]axios[\\/]/ },
]

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/v1': {
        target: 'http://localhost:8099',
        changeOrigin: true,
      },
      // 内嵌 IAM（mode=op）托管在 /iam（/iam/config.json、/iam/oidc、/iam/install、内嵌登录页等）。
      // 仅 op 模式需要；rp 模式前端直连外部 VITE_OIDC_ISSUER，无需本站 /iam 代理（保留无害）。
      //
      // 注意：vite 的 proxy 用对象 key 前缀匹配（plain key 走 `url.startsWith(key)`），并按
      // Object.keys 的**插入顺序**逐条命中（第一个命中即止），不会自动「长 key 优先」。
      // 因此 `/iam/login` 的 GET-only bypass 必须排在更宽泛的 `/iam` 之前，否则会被 `/iam` 抢先代理。
      //
      // GET /iam/login 是浏览器导航（OP 把用户送回 SPA 账密页），应返回 index.html 由前端渲染；
      // 而 POST /iam/oidc/login（登录提交）等非 GET 请求仍须代理到后端，故 bypass 只对 GET 生效。
      '/iam/login': {
        target: 'http://localhost:8099',
        changeOrigin: true,
        bypass: (req) => (req.method === 'GET' ? '/index.html' : undefined),
      },
      '/iam': {
        target: 'http://localhost:8099',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined
          const hit = VENDOR_CHUNKS.find((chunk) => chunk.test.test(id))
          if (hit) return hit.name
          // 其余零散依赖（@babel/runtime、stylis、throttle-debounce 等）归入通用 vendor
          return 'vendor'
        },
      },
    },
  },
})
