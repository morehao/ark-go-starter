# 前端代码规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)
>
> **前端视觉/设计规范的事实源是 [`docs/DESIGN.md`](../../docs/DESIGN.md)**（Ant Design 原生风格全文，遵循 Google DESIGN.md 官方格式规范，含配色/排版/间距/圆角/层级/组件规格/Do's and Don'ts）。可用 `npx @google/design.md lint docs/DESIGN.md` 校验。
> 本文只写**工程约定 + 硬规则索引**，不复制任何设计值 —— 避免 `theme.ts` / `docs/DESIGN.md` / 本文三处漂移。

## 事实源与同步链

```
packages/ui/src/theme.ts 的 tokens  ←(唯一代码事实源)→  docs/DESIGN.md front matter  ←→  页面实际取值
```

- 改任何色值 / 圆角 / 间距：**先改 `tokens`，再同步 `docs/DESIGN.md` front matter**，顺序不可颠倒。
- 二者为**人工同步**；`docs/DESIGN.md` 可用 `npx @google/design.md lint docs/DESIGN.md` 做结构/对比度校验，但 token 具体取值仍需人工核对两处。
- 页面与组件一律引用 `tokens` 或 antd token，**禁止出现裸色值**。

## 技术栈与包结构

仓库根是 pnpm monorepo（workspace 成员为 `apps/web`、`packages/*`、`tests/e2e`）：

| 路径 | 包名 | 作用 |
|------|------|------|
| `packages/ui/` | `@ark-go-starter/ui` | 主题与通用组件：`themeConfig` / `tokens` / `AppShell` / `PageContainer` |
| `packages/api/` | `@ark-go-starter/api` | 接口封装 |
| `packages/types/` | `@ark-go-starter/types` | 共享类型 |
| `packages/tsconfig/` | `@ark-go-starter/tsconfig` | 共享 TS 配置 |
| `apps/web/` | `@ark-go-starter/demo-web` | 演示前端应用（Vite + React + AntD），dev :3000，代理 `/v1` → `http://localhost:8099` |

统一入口：

```tsx
import { AppShell, PageContainer, tokens } from '@ark-go-starter/ui'
```

组件内取 antd 语义 token 用 `theme.useToken()`。

## 命令

在**仓库根**执行（pnpm workspace 根与根 `package.json` 均在此）：

| 目的 | 命令 | 根目录等价 |
|------|------|------------|
| 启动开发服务 | `pnpm dev` | `make dev-frontend` |
| 构建 | `pnpm build` | — |
| 类型检查 | `pnpm typecheck` | — |
| ESLint | `pnpm lint`（修复：`pnpm lint:fix`） | `make lint-frontend` |
| 前后端一起起 | — | `make dev-all APP=demo` |
| 停止前端服务 | — | `make stop-frontend` |

> `pnpm dev` 会 `--filter @ark-go-starter/demo-web`；`pnpm build` 按 `packages/*` → `apps/*` 顺序递归构建。

## 硬性规则（必须遵守）

| # | 规则 | 详情位置 |
|---|------|----------|
| 1 | 每个 app 的 `main.tsx` **必须**用 `AppShell` 包裹（全局主题唯一挂载点：`ConfigProvider` + antd `App` + 根容器 `tabular-nums`） | [DESIGN.md §Components](../../docs/DESIGN.md#components) |
| 2 | 所有管理页用 `PageContainer` 包裹（`title` 必填），**页面不要再重复套 Card** | [DESIGN.md §Layout](../../docs/DESIGN.md#layout) |
| 3 | 禁止硬编码色值（如主色 `#1677ff`、边框 `#f0f0f0`、文字 `rgba(0,0,0,0.65)` 等）；一律走 `tokens` | [DESIGN.md §Do's and Don'ts](../../docs/DESIGN.md#dos-and-donts) |
| 4 | 主色蓝 `primary` 是**稀有资源**：只用于主按钮/链接/选中态/焦点环/分页当前页；页面表面一律中性灰阶，禁止主色淡底 tint 铺满 | [DESIGN.md §Colors](../../docs/DESIGN.md#colors) |
| 5 | 卡片与内容区 **hairline-only 无阴影**；阴影只给 Modal / Dropdown / message 等浮层 | [DESIGN.md §Elevation & Depth](../../docs/DESIGN.md#elevation--depth) |
| 6 | Table 一律 `tableLayout="fixed"` + **每列显式列宽** + `scroll={{ x: 列宽之和 }}`；长文本省略号截断 | [DESIGN.md §Layout](../../docs/DESIGN.md#layout) |
| 7 | 禁止新增 css 文件承载后台样式，样式以 inline style + tokens 表达 | [DESIGN.md §Components](../../docs/DESIGN.md#components) |
| 8 | 状态色走语义 token，禁止用传统色名 Tag（`color="green"/"red"`） | [DESIGN.md §Do's and Don'ts](../../docs/DESIGN.md#dos-and-donts) |
| 9 | 左右分栏用原生 flex（`display:flex; gap:16`，左栏 `flexShrink:0`、右栏 `flex:1`），**禁止用 `<Space>` 依赖内层 `flex:1` 撑满** | [DESIGN.md §Layout](../../docs/DESIGN.md#layout) |
| 10 | 删除无引用 import，避免 `noUnusedLocals` 报错 | [DESIGN.md §Do's and Don'ts](../../docs/DESIGN.md#dos-and-donts) |

## 新增页面检查清单

1. 页面用 `PageContainer` 包裹，不额外套 Card；
2. 颜色/圆角/间距全部取 `tokens` 或 `theme.useToken()`，无裸值；
3. 列表页按「搜索区 → Table → Modal/Drawer」顺序组织，列宽与 `scroll.x` 对齐；
4. 没有新增 `.css` 文件、没有 pill 形按钮、卡片无阴影；
5. 跑一次 `pnpm typecheck` 确认无未使用变量与类型错误；
6. 跑一次 `pnpm lint`（或根目录 `make lint-frontend`），必须 0 error。

## 已知空白

- `packages/ui` 目前仅落地 `theme` / `AppShell` / `PageContainer` / `StatCard` / `StatRow`；状态 Tag 字典、列宽工厂（`idColumn` / `timeColumn` / `actionColumn`）、`TimeCell` / `IDCell` / `RemoteSelect` 待按业务引入；
- demo 已落地多页骨架：深色 `Sider`（`#001529`）+ 顶部 `Header`（64px）+ `Content`；当前菜单有**商品管理 / 数据字典 / 配置中心 / 定时任务 / 异步任务 / 健康检查 / 文件上传**七个（对应 `apps/web/src/pages/{Products,Dicts,Settings,Cron,Async,Health,Files}`，切页仍用 `antd` `Menu` + `useState`）；`apps/web/src/auth/` 已落地 IAM 登录/provider 装配（`config.ts` + `index.tsx`，含 `react-router-dom` 的 `BrowserRouter`），业务页切换本身仍未走路由库；
- IAM 登录页已落地（`@morehao/ark-iam-react/login`，op/rp/disabled 三态，见 [iam.md](./iam.md)）；`apps/web/src/main.tsx` 用 `IAMAuthProvider` + `AuthGuard` 包裹；
- 尚未接入状态管理库；如需业务页 URL 可定位/可回退的完整前端路由，先补 `AppShell` 之外的 `MainLayout` 规范再落地；
- 引入上述内容时，同步更新 `docs/DESIGN.md`，本文的空白清单也应一并核对。

## 相关文档

- 前端视觉规范全文：[docs/DESIGN.md](../../docs/DESIGN.md)
- 后端代码规范：[backend-conventions.md](./backend-conventions.md)
- 命令大全：[commands.md](./commands.md)
