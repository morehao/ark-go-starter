# 前端代码规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)
>
> **前端视觉/设计规范的事实源是 [`frontend/DESIGN.md`](../../frontend/DESIGN.md)**（「冷白工程台」设计语言全文，228 行，含色彩/排版/间距/圆角/组件规格/Do's and Don'ts）。
> 本文只写**工程约定 + 硬规则索引**，不复制任何设计值 —— 避免 `theme.ts` / `DESIGN.md` / 本文三处漂移。

## 事实源与同步链

```
packages/ui/src/theme.ts 的 tokens  ←(唯一代码事实源)→  frontend/DESIGN.md front matter  ←→  页面实际取值
```

- 改任何色值 / 圆角 / 间距：**先改 `tokens`，再同步 `frontend/DESIGN.md` front matter**，顺序不可颠倒。
- 二者为**人工同步**，项目内暂无 lint/test 强制一致（见 DESIGN.md `Known Gaps`），因此每次改动都必须人工核对两处。
- 页面与组件一律引用 `tokens` 或 antd token，**禁止出现裸色值**。

## 技术栈与包结构

`frontend/` 是 pnpm monorepo：

| 路径 | 包名 | 作用 |
|------|------|------|
| `packages/ui/` | `@ark-go-starter/ui` | 主题与通用组件：`themeConfig` / `tokens` / `AppShell` / `PageContainer` |
| `packages/api/` | `@ark-go-starter/api` | 接口封装 |
| `packages/types/` | `@ark-go-starter/types` | 共享类型 |
| `packages/tsconfig/` | `@ark-go-starter/tsconfig` | 共享 TS 配置 |
| `apps/demo-web/` | `@ark-go-starter/demo-web` | 演示前端应用（Vite + React + AntD），dev :3000，代理 `/v1` → `http://localhost:8099` |

统一入口：

```tsx
import { AppShell, PageContainer, tokens } from '@ark-go-starter/ui'
```

组件内取 antd 语义 token 用 `theme.useToken()`。

## 命令

在 `frontend/` 目录下（或在根目录用 Makefile 等价命令）：

| 目的 | 命令 | 根目录等价 |
|------|------|------------|
| 启动开发服务 | `pnpm dev` | `make dev-frontend` |
| 构建 | `pnpm build` | — |
| 类型检查 | `pnpm typecheck` | — |
| 前后端一起起 | — | `make dev-all APP=demo` |
| 停止前端服务 | — | `make stop-frontend` |

> `pnpm dev` 会 `--filter @ark-go-starter/demo-web`；`pnpm build` 按 `packages/*` → `apps/*` 顺序递归构建。

## 硬性规则（必须遵守）

| # | 规则 | 详情位置 |
|---|------|----------|
| 1 | 每个 app 的 `main.tsx` **必须**用 `AppShell` 包裹（全局主题唯一挂载点：`ConfigProvider` + antd `App` + 根容器 `tabular-nums`） | [DESIGN.md §7.1](../../frontend/DESIGN.md#71-全局骨架appshell) |
| 2 | 所有管理页用 `PageContainer` 包裹（`title` 必填），**页面不要再重复套 Card** | [DESIGN.md §7.2](../../frontend/DESIGN.md#72-页面容器pagecontainer) |
| 3 | 禁止硬编码色值（`#4f6ef7`、`#e6e9ef`、`#94a3b8`、`rgba(...)` 品牌投影等） | [DESIGN.md §8 Don't](../../frontend/DESIGN.md#8-dos-and-donts) |
| 4 | 主色靛蓝是**稀有资源**：只用于主按钮/链接/选中态/焦点环/品牌区；页面表面一律中性灰阶，禁止主色淡底 tint | [DESIGN.md §2](../../frontend/DESIGN.md#2-色彩) |
| 5 | 卡片与内容区 **hairline-only 无阴影**；阴影只给 Modal / Dropdown / Drawer 等浮层 | [DESIGN.md §5](../../frontend/DESIGN.md#5-层级与阴影) |
| 6 | Table 一律 `tableLayout="fixed"` + **每列显式列宽** + `scroll={{ x: 列宽之和 }}`；长文本省略号截断 | [DESIGN.md §7.3](../../frontend/DESIGN.md#73-列表页模板) |
| 7 | 禁止新增 css 文件承载后台样式，样式以 inline style + tokens 表达 | [DESIGN.md §7.1](../../frontend/DESIGN.md#71-全局骨架appshell) |
| 8 | 状态色走语义 token，禁止用传统色名 Tag（`color="green"/"red"`） | [DESIGN.md §8 Don't](../../frontend/DESIGN.md#8-dos-and-donts) |
| 9 | 左右分栏用原生 flex（`display:flex; gap:16`，左栏 `flexShrink:0`、右栏 `flex:1`），**禁止用 `<Space>` 依赖内层 `flex:1` 撑满** | [DESIGN.md §4](../../frontend/DESIGN.md#4-布局与间距) |
| 10 | 删除无引用 import，避免 `noUnusedLocals` 报错 | [DESIGN.md §8 Do](../../frontend/DESIGN.md#8-dos-and-donts) |

## 新增页面检查清单

1. 页面用 `PageContainer` 包裹，不额外套 Card；
2. 颜色/圆角/间距全部取 `tokens` 或 `theme.useToken()`，无裸值；
3. 列表页按「搜索区 → Table → Modal/Drawer」顺序组织，列宽与 `scroll.x` 对齐；
4. 没有新增 `.css` 文件、没有 pill 形按钮、卡片无阴影；
5. 跑一次 `pnpm typecheck` 确认无未使用变量与类型错误。

## 已知空白（与 DESIGN.md `Known Gaps` 一致）

- `packages/ui` 目前仅落地 `theme` / `AppShell` / `PageContainer`；状态 Tag 字典、列宽工厂（`idColumn` / `timeColumn` / `actionColumn`）、`TimeCell` / `IDCell` / `RemoteSelect` 待按业务引入；
- demo 未使用侧栏与登录页，相关令牌（`surface-sidebar` / `selected-bg` / 渐变）为模板预留；
- 未接入路由库 / 状态管理库；如需多页应用，先补 `AppShell` 之外的 `MainLayout` 规范再落地；
- 引入上述内容时，同步更新 `frontend/DESIGN.md`，本文的空白清单也应一并核对。

## 相关文档

- 前端设计语言全文：[frontend/DESIGN.md](../../frontend/DESIGN.md)
- 后端代码规范：[backend-conventions.md](./backend-conventions.md)
- 命令大全：[commands.md](./commands.md)
