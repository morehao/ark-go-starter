---
version: 1.0
name: ark-go-starter-frontend-design
description: "ark-go-starter 前端统一视觉语言。方向为「冷白工程台」（方法论受 Stripe / Vercel 工程控制台启发）：靛蓝 #4f6ef7 是唯一强调色，仅用于主按钮 / 链接 / 选中态 / 品牌区；深蓝黑 #0f172a 侧栏；页面表面一律中性灰阶（禁用靛蓝淡底 tint）；卡片与内容区用冷调 hairline 边框分层、不使用卡片阴影（阴影仅留给 Modal 等浮层）；文字为三级 hex 灰阶并全站开启 tabular-nums（ID/时间/数字等宽）。代码事实源：packages/ui/src/theme.ts 的 tokens 对象（本文件与之一一对应，改动需同步两处）。"

colors:
  brand-primary: "#4f6ef7"
  brand-primary-hover: "#6b86ff"
  brand-primary-active: "#3a55d6"
  brand-purple: "#7a5af8"
  brand-gradient: "linear-gradient(135deg, #4f6ef7 0%, #7a5af8 55%, #a855f7 100%)"
  brand-gradient-soft: "linear-gradient(135deg, #eef2ff 0%, #f5f0ff 100%)"
  surface-layout: "#f6f7f9"
  surface-card: "#ffffff"
  surface-sidebar: "#0f172a"
  surface-table-header: "#f7f8fa"
  surface-table-row-hover: "#f3f4f6"
  surface-soft-fill: "#f1f3f5"
  surface-selected-bg: "rgba(79, 110, 247, 0.14)"
  surface-code: "#fafafa"
  border-hairline: "#e6e9ef"
  border-strong: "#d3d8e0"
  text-primary: "#1f2430"
  text-secondary: "#64748d"
  text-placeholder: "#94a3b8"
  semantic-success: "#22c55e"
  semantic-warning: "#f59e0b"
  semantic-error: "#ef4444"
  semantic-warning-bg: "#fffbe6"
  semantic-warning-border: "#ffe58f"

typography:
  family: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
  body-size: 14px
  mono: "ui-monospace, SFMono-Regular, Menlo, monospace"

rounded:
  control: 8px
  card: 12px
  modal: 14px

spacing:
  base: 4px
  page-content-margin: 20px
  content-padding: 20px
  search-control-width: 240px

components:
  main-header:
    backgroundColor: "{colors.surface-card}"
    height: 56px
  page-card:
    backgroundColor: "{colors.surface-card}"
    border: "1px solid {colors.border-hairline}"
    rounded: "{rounded.card}"
    body-padding: "{spacing.content-padding}"
  table-header:
    backgroundColor: "{colors.surface-table-header}"
    color: "{colors.text-secondary}"
  table-row-hover: "{colors.surface-table-row-hover}"
---

# ark-go-starter 前端 DESIGN.md

> 供 AI 编码代理与开发者共同遵循的前端「统一风格约定」。
> **代码唯一事实源是 `packages/ui/src/theme.ts` 的 `tokens` 对象**：改任何色值/圆角/间距必须先改 tokens，再同步本文件 front matter（二者必须一致）。业务页面与组件一律引用 `tokens` / antd token，禁止出现 `#4f6ef7`、`#e6e9ef`、`#94a3b8` 之类的裸色值。

## 1. 视觉主题总览

- **定位**：工程实践模板的前端基座，「冷白工程台」——浅色、数据密集、克制。
- **画布**：页面底色 `{colors.surface-layout}`（冷白）；内容承载于白色圆角卡片 `{colors.surface-card}`，卡片以 `1px solid {colors.border-hairline}` hairline 描边分层、**不使用卡片阴影**（阴影仅留给 Modal 等浮层）。
- **品牌**：靛蓝 `{colors.brand-primary}` 是唯一强调色（主按钮 / 链接 / 选中态 / 输入聚焦）；渐变 `{colors.brand-gradient}` 仅出现在品牌区与 Logo/头像（全站唯一「hero 大气层」）。
- **导航**：约定左侧深色栏 `{colors.surface-sidebar}`（#0f172a，近黑蓝），菜单选中项用 `{colors.surface-selected-bg}` 柔和半透明高亮（非整块亮色）；顶部白 Header 56px。当前 demo 仅有 Header，侧栏为模板规范预留。
- **基调**：以「表格 + 搜索 + 抽屉/弹窗」构成业务页；页面表面一律**中性灰阶**（禁用靛蓝淡底 tint），把主色当作稀有资源；全站开启 `tabular-nums` 让 ID/时间/数字等宽对齐。

> **参照锚点**：方法论受 **Stripe**（数据表格骨架、`tabular-nums`、ink-mute 灰阶文字）与 **Vercel**（精确灰阶分层、渐变只出现在品牌区这一个「hero 大气层」）启发，纪律细节参考 **Linear / Supabase**（主色稀有、hairline 层级）。非照搬任何具体品牌配色。

## 2. 色彩

### 品牌与强调

| 令牌 | 值 | 用途 | 稀缺级别 |
|---|---|---|---|
| `{colors.brand-primary}` | #4f6ef7 | 主按钮、链接、菜单选中、焦点环、品牌图标 | **稀有，只用这三五处** |
| `{colors.brand-primary-hover}` | #6b86ff | 主按钮 hover | 继承 |
| `{colors.brand-primary-active}` | #3a55d6 | 主按钮按下 | 继承 |
| `{colors.brand-purple}` | #7a5af8 | 分类图标强调（统计卡等） | 装饰点缀 |
| `{colors.brand-gradient}` | 靛→紫渐变 | Logo、头像、品牌区 | 品牌专属 |
| `{colors.brand-gradient-soft}` | 淡靛渐变 | 品牌区浅淡底（页面内禁用） | 轻量 |

**禁止**：把主色/渐变大面积铺满页面、作任意卡片默认背景、作正文颜色；页面内不得出现主色淡底 tint（表头/行 hover/卡片一律中性）。

### 表面 / 边框 / 文字

| 组 | 令牌 | 值 | 说明 |
|---|---|---|---|
| 表面 | `{colors.surface-layout}` | #f6f7f9 | 页面布局底色（antd colorBgLayout），冷白 |
| | `{colors.surface-card}` | #ffffff | 卡片 / 表头底（Header） |
| | `{colors.surface-sidebar}` | #0f172a | 侧栏深底（预留） |
| | `{colors.surface-table-header}` | #f7f8fa | Table 表头（中性，禁止靛 tint） |
| | `{colors.surface-table-row-hover}` | #f3f4f6 | Table 行 hover（中性） |
| | `{colors.surface-soft-fill}` | #f1f3f5 | 中性填充底（统计卡图标底等） |
| | `{colors.surface-code}` | #fafafa | 代码/内容块底 |
| 边框 | `{colors.border-hairline}` | #e6e9ef | 冷调 hairline，卡片/分隔（antd colorBorderSecondary 同源） |
| | `{colors.border-strong}` | #d3d8e0 | 更强的分割线（Header 内竖分割线等） |
| 文字 | `{colors.text-primary}` | #1f2430 | 冷近黑正文/标题（antd colorText） |
| | `{colors.text-secondary}` | #64748d | 次要说明（antd colorTextSecondary，借鉴 Stripe ink-mute） |
| | `{colors.text-placeholder}` | #94a3b8 | 空态、占位、弱辅助文字 |

### 语义色

`{colors.semantic-success}` `#22c55e` / `{colors.semantic-warning}` `#f59e0b` / `{colors.semantic-error}` `#ef4444`，与 antd `colorSuccess/colorWarning/colorError` 对齐，仅用于状态 Tag / 告警。轻告警底 `{colors.semantic-warning-bg}` #fffbe6、边 `{colors.semantic-warning-border}` #ffe58f。

## 3. 字体与排版

- 字体栈见 front matter `typography.family`（antd fontFamily 已全局注入），中文优先 PingFang SC / Microsoft YaHei。
- 字号沿用 antd 层级：正文 14px（默认）、标题用 antd `Typography`、表内弱文字 12–13px、mono 13px。
- **等宽字体场景**：ID/UUID、Key/Secret、代码/日志 payload、菜单路径 `{typography.mono}`。
- 全站根容器开启 `font-variant-numeric: tabular-nums`（由 `AppShell` 注入），ID/时间/数字等宽对齐。
- 表格中辅助说明使用 `{colors.text-secondary}`；空态与弱说明（「加载中… / 未设置」等）使用 `{colors.text-placeholder}`。

## 4. 布局与间距

- 间距基数 4px；管理页由 `PageContainer` 统一骨架：页标题 + 描述 + 右上操作区（extra），下方白色内容卡（body padding 20、圆角 12、hairline 边）。
- **左右分栏**（固定宽左栏 + 弹性右栏）用原生 flex 容器：`display:flex; alignItems:flex-start; gap:16`，左栏 `flexShrink:0` 定宽、右栏 `flex:1` 撑满剩余宽度。**禁止用 antd `<Space>` 包裹左右卡片并依赖内层 `flex:1` 撑满**：`Space` 会把子项再包一层不参与 grow 的 `.ant-space-item`，右卡片不会拉伸。
- 布局内边距：`Content` 外边距 20，卡片间距 16–20。
- 搜索区：`Input.Search allowClear`，宽度统一 `{spacing.search-control-width}`（240）。
- Table：`rowKey`、`loading`、分页 `showSizeChanger + showTotal: (t) => \`共 ${t} 条\``；操作列放最右。

## 5. 层级与阴影

- **卡片与内容区一律 hairline-only，无阴影**：层间靠 surface 灰阶 + 冷调 hairline 边框 + 留白分层（Linear / Supabase 哲学）。阴影只留给浮层：Modal / Dropdown / Drawer 使用 antd `boxShadowSecondary = 0 6px 24px rgba(15,23,42,0.08)`。
- 侧栏（Sider）紧贴画布靠色彩对比分层，**不加投影**；Header 用底部 hairline 分割。
- 深色底上的玻璃元素（半透明白 rgba 层）只存在于品牌区。

## 6. 圆角

| 场景 | 圆角 |
|---|---|
| 按钮 / 输入 / 单元格 Tag / Menu 项 | 8px（antd `borderRadius`） |
| 页面卡片 / 内容卡 / 详情 | 12px（antd `borderRadiusLG`） |
| Modal | 14px |

**禁止**胶囊（pill）形按钮；Tag 可用默认小圆角。

## 7. 组件规格

### 7.1 全局骨架（AppShell）

- 每个前端 app 的 `main.tsx` **必须**用 `AppShell`（`@ark-go-starter/ui`）包裹，它是全局主题唯一挂载点：`ConfigProvider locale=zhCN theme=themeConfig` + antd `App` + 根容器 `tabular-nums`。
- Header 白底 56px（`themeConfig.components.Layout.headerHeight`），标题用 `{colors.text-primary}`；不要给 Header 手写深色底或白色文字。
- **禁止各 app 新增 css 文件承载后台样式**：样式以 inline style + tokens 表达。

### 7.2 页面容器（PageContainer）

- 所有管理页用 `PageContainer` 包裹：`title`（必填）+ `description` + `extra`（右上操作区，如刷新、主操作）+ 内容。
- 内容卡由组件内建（白底、hairline 边、12px 圆角、20px 内边距），页面不要重复套 Card。

### 7.3 列表页模板

按以下顺序书写，保证各列表页观感一致：

```
PageContainer(title, description, extra=刷新 + 主操作[type=primary])
  ├─ 搜索区（Input.Search 240 / Select 筛选）
  ├─ Table(rowKey, loading, tableLayout="fixed", scroll={{ x: 列宽之和 }}, pagination.showTotal + showSizeChanger)
  └─ 新建/编辑 Modal（layout="vertical" + confirmLoading）
     详情 Drawer（Descriptions bordered size="small"）
```

**列宽硬规则**

- **所有列都必须有显式列宽**，`scroll.x` 取各列宽之和（禁止写与内容无关的整百「凑数宽度」）；同类字段在各页应同宽。
- **列宽必须严格生效**：Table 一律 `tableLayout="fixed"`；auto 布局下列宽只是建议值，长文本会把列撑开、出现本可避免的横向滚动条。
- 文本列超长时用省略号截断并悬浮展示全文，不让长文本换行把行高撑成两行。

## 8. Do's and Don'ts

### Do

- 改设计值：先改 `packages/ui/src/theme.ts` 的 `tokens`，再同步本文件 front matter（两处一致）。
- 页面/组件引用颜色一律用 `tokens.*`（`import { tokens } from '@ark-go-starter/ui'`）或 antd `theme.useToken()`。
- 页面用 `PageContainer` 骨架；列表 Table 用 `tableLayout="fixed"` + 显式列宽 + `scroll={{ x }}`。
- 主色仅用于主操作/链接/选中/焦点；页面表面保持中性灰阶 + 白卡（冷白工程台）。
- 卡片 hairline-only 无阴影；阴影只给 Modal/浮层。
- 左右分栏用原生 flex 容器（见 §4）。
- 删除无引用 import，避免 `noUnusedLocals` 报错。

### Don't

- 禁止在业务代码硬编码任何色值（示例：`#4f6ef7`、`#e6e9ef`、`#94a3b8`、`rgba(...)` 品牌投影等）。
- 禁止用传统色名 Tag（`color="green"/"red"/"orange"`）表示状态，状态色一律走语义 token。
- 禁止 pill 形按钮；禁止把主色渐变当卡片默认背景。
- 禁止在页面表面使用主色淡底 tint（如 `#ece9ff`、`#fafbff`、`#f6f8ff` 一类表头/行 hover/卡片）——一律中性灰阶。
- 禁止在列表页手写裸列宽数字却省略 `tableLayout="fixed"`（列宽会失效、长文本会撑列）。
- 禁止新增 css 文件承载后台样式；样式以 inline style + tokens 表达。

## 9. 响应式

- 管理台桌面优先：页面级断点行为不单独处理，宽度不足时由 `scroll={{ x }}` + `tableLayout="fixed"` 产生横向滚动。
- 最小宽度建议 ≥ 1100px；栅格 `Row/Col` 需要时用 `xs/sm/lg`。

## 10. 令牌清单速查（与 theme.ts tokens 对齐）

| 代码 tokens | DESIGN 令牌 | 语义 |
|---|---|---|
| `tokens.primary` | brand-primary | 主强调色 |
| `tokens.primaryHover` / `primaryActive` | brand-primary-hover / active | 主色交互态 |
| `tokens.purple` | brand-purple | 分类紫 |
| `tokens.gradient` / `gradientSoft` | brand-gradient(-soft) | 品牌渐变 |
| `tokens.bg` | surface-layout | 页面底 |
| `tokens.cardBg` / `headerBg` | surface-card | 卡/Header 底 |
| `tokens.sidebarBg` | surface-sidebar | 侧栏底（预留） |
| `tokens.tableHeaderBg` / `rowHoverBg` | surface-table-header / row-hover | 表格（中性） |
| `tokens.softFill` | surface-soft-fill | 中性填充底 |
| `tokens.selectedBg` | surface-selected-bg | 深色导航选中柔和底 |
| `tokens.codeBg` | surface-code | 代码块底 |
| `tokens.border` / `borderStrong` | border-hairline / border-strong | 边框 |
| `tokens.text` / `textSecondary` / `textPlaceholder` | text-primary / secondary / placeholder | 文字三级 |
| `tokens.success` / `warning` / `error` | semantic-success / warning / error | 语义色 |
| `tokens.warningBg` / `warningBorder` | semantic-warning-bg / warning-border | 语义告警淡底 |

## Known Gaps

1. `theme.ts` 与 DESIGN.md 为**人工同步**，暂无 lint/test 强制一致。
2. 当前 `packages/ui` 仅落地 `theme` / `AppShell` / `PageContainer` 三项；状态 Tag 字典、列宽工厂（`idColumn` / `timeColumn` / `actionColumn`）、`TimeCell`、`IDCell`、`RemoteSelect` 等按业务需要再逐步引入，并同步补充本文件。
3. demo 未使用侧栏与登录页，相关令牌（`surface-sidebar` / `selected-bg` / 渐变）暂作为模板规范预留。
4. 未接入路由库/状态管理库；如需多页应用，先补 `AppShell` 之外的 `MainLayout` 规范再落地。
