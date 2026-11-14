---
version: alpha
name: Aperture Dashboard — Ant Design Native
description: "ark-go-starter 前端统一视觉语言 · Ant Design 原生风格。以 antd 5.x 默认设计语言为基线：蓝色主色 #1677ff、深色 Sider #001529 + 浅色 Layout、6px 控件圆角 / 8px 卡片圆角、信息密度高、组件成熟。本文档是项目前端视觉规范的唯一事实源；代码事实源为 packages/ui/src/theme.ts 的 tokens（本文件与之一一对应，改动需两处同步）。遵循 Google DESIGN.md 官方格式规范（google-labs-code/design.md），可用 npx @google/design.md lint docs/DESIGN.md 校验。"

colors:
  primary: "#1677ff"
  primary-strong: "#0b63d6"
  primary-hover: "#4096ff"
  primary-active: "#0958d9"
  primary-soft: "#e6f4ff"
  primary-soft-border: "#91caff"
  secondary: "#4096ff"
  tertiary: "#69b1ff"
  surface: "#ffffff"
  surface-layout: "#f5f5f5"
  surface-sider: "#001529"
  surface-sider-sub: "#000c17"
  surface-table-header: "#fafafa"
  surface-fill: "#f5f5f5"
  border: "#f0f0f0"
  border-strong: "#d9d9d9"
  on-surface: "rgba(0,0,0,0.88)"
  on-surface-secondary: "rgba(0,0,0,0.65)"
  on-surface-tertiary: "rgba(0,0,0,0.45)"
  on-surface-quaternary: "rgba(0,0,0,0.25)"
  success: "#52c41a"
  warning: "#faad14"
  error: "#ff4d4f"
  tag-success-bg: "#f6ffed"
  tag-success-border: "#b7eb8f"
  tag-success-text: "#237804"
  tag-warning-bg: "#fffbe6"
  tag-warning-border: "#ffe58f"
  tag-warning-text: "#874d00"
  tag-error-bg: "#fff2f0"
  tag-error-border: "#ffccc7"
  tag-error-text: "#a8071a"
  tag-default-bg: "#fafafa"
  tag-default-border: "#d9d9d9"

typography:
  page-title:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 20px
    fontWeight: 600
    lineHeight: 1.4
  card-title:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 16px
    fontWeight: 600
    lineHeight: 1.5
  statistic-value:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 26px
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: -0.3px
  body:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.5715
  label:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 14px
    fontWeight: 600
    lineHeight: 1.5
  small:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.5
  tag:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif"
    fontSize: 12px
    fontWeight: 400
    lineHeight: 20px

rounded:
  none: 0px
  sm: 4px
  md: 6px
  lg: 8px
  full: 9999px

spacing:
  base: 4px
  content-padding: 24px
  card-padding: 20px
  grid-gap: 16px
  control-height: 32px
  header-height: 64px
  sider-width: 208px
  cell-padding-y: 12px
  cell-padding-x: 16px

components:
  sider:
    backgroundColor: "{colors.surface-sider}"
    textColor: "rgba(255,255,255,0.65)"
    width: "{spacing.sider-width}"
    height: "{spacing.header-height}"
  menu-item:
    textColor: "rgba(255,255,255,0.65)"
    rounded: "{rounded.md}"
    height: "40px"
    padding: "0 24px"
  menu-item-active:
    backgroundColor: "{colors.primary-strong}"
    textColor: "{colors.surface}"
  header:
    backgroundColor: "{colors.surface}"
    height: "{spacing.header-height}"
    padding: "0 {spacing.content-padding}"
  card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.card-padding}"
  statistic-card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.card-padding}"
  button-primary:
    backgroundColor: "{colors.primary-strong}"
    textColor: "{colors.surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
    padding: "0 15px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
  button-default:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
    padding: "0 15px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"
  table:
    backgroundColor: "{colors.surface}"
  table-header:
    backgroundColor: "{colors.surface-table-header}"
    textColor: "{colors.on-surface-secondary}"
    padding: "{spacing.cell-padding-y} {spacing.cell-padding-x}"
  table-row-hover:
    backgroundColor: "{colors.surface-table-header}"
  tag-success:
    backgroundColor: "{colors.tag-success-bg}"
    textColor: "{colors.tag-success-text}"
    rounded: "{rounded.sm}"
  tag-warning:
    backgroundColor: "{colors.tag-warning-bg}"
    textColor: "{colors.tag-warning-text}"
    rounded: "{rounded.sm}"
  tag-error:
    backgroundColor: "{colors.tag-error-bg}"
    textColor: "{colors.tag-error-text}"
    rounded: "{rounded.sm}"
  tag-default:
    backgroundColor: "{colors.tag-default-bg}"
    textColor: "{colors.on-surface-secondary}"
    rounded: "{rounded.sm}"
  dropdown:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "4px"
  modal:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
  switch-on:
    backgroundColor: "{colors.primary}"
    rounded: "{rounded.full}"
    width: "44px"
    height: "22px"
  switch-off:
    backgroundColor: "rgba(0,0,0,0.25)"
    rounded: "{rounded.full}"
    width: "44px"
    height: "22px"
  pagination-item:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    size: "{spacing.control-height}"
  pagination-item-active:
    backgroundColor: "{colors.primary-strong}"
    textColor: "{colors.surface}"

  badge-success:
    backgroundColor: "{colors.tag-success-bg}"
    textColor: "{colors.tag-success-text}"
  badge-warning:
    backgroundColor: "{colors.tag-warning-bg}"
    textColor: "{colors.tag-warning-text}"
  sidebar-footer:
    backgroundColor: "{colors.surface-sider-sub}"
  page-canvas:
    backgroundColor: "{colors.surface-layout}"
  card-outline:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface-secondary}"
    padding: "{spacing.card-padding}"
  dropdown-item-hover:
    backgroundColor: "{colors.surface-fill}"
    textColor: "{colors.on-surface}"
  button-primary-active:
    backgroundColor: "{colors.primary-active}"
    textColor: "{colors.surface}"
  filter-chip-selected:
    backgroundColor: "{colors.primary-soft}"
    textColor: "{colors.primary-active}"
  table-cell:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    padding: "{spacing.cell-padding-y} {spacing.cell-padding-x}"
  tag-success-outline:
    backgroundColor: "{colors.tag-success-bg}"
    textColor: "{colors.tag-success-text}"
    rounded: "{rounded.sm}"
  tag-warning-outline:
    backgroundColor: "{colors.tag-warning-bg}"
    textColor: "{colors.tag-warning-text}"
    rounded: "{rounded.sm}"
  tag-error-outline:
    backgroundColor: "{colors.tag-error-bg}"
    textColor: "{colors.tag-error-text}"
    rounded: "{rounded.sm}"
  tag-default-outline:
    backgroundColor: "{colors.tag-default-bg}"
    textColor: "{colors.on-surface-secondary}"
    rounded: "{rounded.sm}"
  divider:
    backgroundColor: "{colors.border}"
    height: "1px"
  input-border:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    height: "{spacing.control-height}"

  input-focus-ring:
    backgroundColor: "{colors.primary-soft-border}"
    rounded: "{rounded.full}"
    size: "2px"
  card-border-strong:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface-tertiary}"
  tag-success-border-ref:
    backgroundColor: "{colors.tag-success-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  tag-warning-border-ref:
    backgroundColor: "{colors.tag-warning-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  tag-error-border-ref:
    backgroundColor: "{colors.tag-error-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  tag-default-border-ref:
    backgroundColor: "{colors.tag-default-border}"
    rounded: "{rounded.sm}"
    size: "1px"
  outline-strong:
    backgroundColor: "{colors.border-strong}"
    height: "1px"
  input-placeholder:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface-quaternary}"
    height: "{spacing.control-height}"
  status-dot-success:
    backgroundColor: "{colors.success}"
    rounded: "{rounded.full}"
    size: "8px"
  status-dot-warning:
    backgroundColor: "{colors.warning}"
    rounded: "{rounded.full}"
    size: "8px"
---

## Overview

Aperture Dashboard 是企业级中后台（B 端管理系统）的「经营数据概览」页，采用 **Ant Design 原生设计语言**——中后台事实标准。整体气质**克制、规整、信息密度高、交互可预期**：色彩以中性灰阶为主，蓝色 `primary` 仅作为功能性强调色用于可交互态与选中态；布局为左侧深色导航（Sider）+ 右侧浅色工作区（Header + Content）；网格以 4px 为间距基数。

目标用户为内部数据分析师与运营人员，页面需在有限屏幕内高频承载指标卡、图表与明细表，因此强调「扫视效率」与「操作一致性」，而非视觉张扬。

## Colors

调色板以中性灰阶为骨架，蓝色 `primary` 为唯一强调色。

- **Primary (#1677ff)**: 主按钮、链接、Tab/分段选中、分页当前页、输入聚焦环。仅用于可交互与选中态，不作大面积装饰。
- **Primary hover/active (#4096ff / #0958d9)**: 主色交互态。
- **Primary soft (#e6f4ff, 边 #91caff)**: 选中 Tag 与批量操作提示条的浅蓝底。
- **Surface (#ffffff) / Surface layout (#f5f5f5)**: 白色卡片承载内容，冷白画布作底，靠 hairline 边框分层。
- **Surface sider (#001529)**: 深色侧边导航底。
- **Border (#f0f0f0) / Border strong (#d9d9d9)**: 前者用于卡片/分隔线，后者用于控件描边。
- **On-surface 四级透明度（88/65/45/25%）**: 正文 / 次要 / 辅助 / 占位，遵循 antd 透明度文字体系。
- **Success/Warning/Error (#52c41a / #faad14 / #ff4d4f)**: 语义状态色，仅用于状态 Tag 与告警。

状态 Tag 一律「浅底 + 浅边 + 深字」三件套（见 front-matter `tag-*` 系列）。

## Typography

字体栈统一使用系统字体（`-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, PingFang SC, Microsoft YaHei`），中文优先 PingFang SC / Microsoft YaHei。正文基准 14px、行高 1.5715（antd 默认）。

- **page-title**: 20px / 600 —— 页面主标题。
- **card-title**: 16px / 600 —— 卡片、表格工具栏标题。
- **statistic-value**: 26px / 600，字距 -0.3px —— KPI 指标数值。
- **body**: 14px / 400 —— 正文与表格单元格。
- **label**: 14px / 600 —— 表头、表单标签、导航菜单。
- **small**: 13px / 400 —— 辅助说明、图例。
- **tag**: 12px / 400 —— 状态标签、角标。

数字与金额列建议开启 `font-variant-numeric: tabular-nums` 保证纵向对齐。

## Layout

采用经典后台三区骨架：**左侧固定 Sider（208px）+ 顶部 Header（64px）+ 主内容 Content**。

- **间距基数 4px**，内容区外边距 24px，卡片间距 16px。
- **KPI 指标条** 4 列等宽栅格（`repeat(4,1fr)`），窄屏（≤1100px）降为 2 列。
- **图表区** 双列 `1.6fr / 1fr`（折线宽、环形窄），窄屏单列。
- **控件高度统一 32px**；内容最大宽度 1320px 居中。
- **表格** 固定布局（`table-layout: fixed`）+ 显式列宽 + 横向 `scroll.x`，单元格内边距 `12px 16px`。

## Elevation & Depth

层次以**扁平描边 + 色阶对比**为主，除浮层外不使用投影：

- 卡片与内容区默认**无阴影**，靠 `border: 1px solid #f0f0f0` + 白底 vs `#f5f5f5` 画布分层。
- 指标卡（statistic-card）仅 hover 时出现轻阴影：`0 3px 12px rgba(0,0,0,0.06)`。
- **浮层**（Modal / Dropdown / message）使用 antd 标准浮层阴影：`0 6px 16px rgba(0,0,0,0.08), 0 3px 6px -4px rgba(0,0,0,0.12), 0 9px 28px 8px rgba(0,0,0,0.05)`。
- 输入聚焦：边框转主色 + 聚焦环 `0 0 0 2px rgba(5,145,255,0.1)`。
- Sider 靠深色底与浅色工作区对比分层，不加投影。

## Shapes

形状语言为**方角 + 小圆角**，体现 antd 的实用工程感：

- 按钮 / 输入框 / 分段 / 分页器 / 菜单项：**6px**（`rounded.md`）。
- 卡片 / Statistic 卡 / Dropdown / Modal：**8px**（`rounded.lg`）。
- 状态 Tag：**4px**（`rounded.sm`）。
- 头像（圆形）、开关（胶囊）：全圆 `rounded.full`。

不使用大圆角或胶囊形按钮（头像、开关除外）。

## Components

- **Buttons**: 主按钮 `button-primary` 主色底白字、6px 圆角、高 32px；次按钮 `button-default` 白底 hairline 描边，hover 转主色；按下用 `button-primary-hover` 色阶。
- **Inputs**: 高 32px、6px 圆角、`#d9d9d9` 描边；hover 主色描边、focus 主色边框 + 聚焦环；占位符 `on-surface-quaternary`。
- **Nav (Sider / Menu)**: 深色底 `#001529`，菜单项高 40px、文字 `rgba(255,255,255,0.65)`；选中项 `menu-item-active` 主色底白字。
- **Table**: 固定布局；表头 `#fafafa` 底 + `on-surface-secondary` 文字 + 600 字重；可排序列右侧 sorter 上下箭头（激活主色）；行 hover `#fafafa`。**操作列一律用 `actionColumn()`**（`fixed: 'right'`，列宽由 `actionColumnWidth()` 计算，禁止手写宽度）：`variant: 'icon'` 为「图标 + 文字」链接按钮，操作数超过 `max` 时其余的收进「更多」下拉；页面不再手写 `Space` + `Button type="link"`。
- **Tag（Chips）**: 状态标签 12px、4px 圆角、行高 20px，四种语义（success/warning/error/default）采用浅底浅边深字。
- **Pagination**: 页码项 32px 方形、6px 圆角；当前页 `pagination-item-active` 主色底白字；右侧「共 N 条」+ 每页条数选择。
- **Modal**: 宽 480px、8px 圆角、浮层阴影；标题栏（16px/600 + ✕）+ 垂直表单 + 底栏（取消 / 确定主按钮）。
- **Switch**: 44×22px 胶囊，开 `#1677ff`、关 `rgba(0,0,0,0.25)`。
- **Message**: 顶部居中白底卡片 + 浮层阴影 + 成功图标（success 色）。

## Do's and Don'ts

- **Do** 颜色/圆角/间距一律引用本文 token（`{colors.*}` / `{rounded.*}` / `{spacing.*}`），落地时映射到 antd `theme.token`。
- **Do** 主色只用于可交互态与选中态；页面表面用中性灰阶 + 白卡分层。
- **Do** 表格用 `table-layout: fixed` + 显式列宽 + `scroll.x`。
- **Do** 状态一律用语义 Tag（浅底浅边深字），保持 WCAG AA 对比度（正文 ≥ 4.5:1）；**可切换**的状态列改用 `StatusSwitch`（双向二次确认、受控、失败自动回滚），并从操作列移除对应的「启用/停用」按钮。
- **Don't** 把主色当大面积背景或正文色。
- **Don't** 使用大圆角 / 胶囊形按钮（头像、开关除外）。
- **Don't** 给卡片加默认投影（仅 hover 与浮层允许）。
- **Don't** 用传统色名字符串（`green`/`red`/`orange`）表达状态，一律走语义 token。
- **Don't** 列宽只写数字而省略固定布局（列宽会失效）。

## Responsive

- 桌面优先，最小推荐宽度 ≥ 1100px。
- 断点 ≤1100px：KPI 4 列 → 2 列；图表区双列 → 单列；Sider 隐藏（可改为抽屉）。
- 表格保持固定布局，宽度不足时横向滚动，不挤压列；控件高度与字号在窄屏不缩小，保证可点性。

## Token → antd 映射

| DESIGN 令牌 | 值 | antd token |
|---|---|---|
| colors.primary | #1677ff | colorPrimary |
| colors.primary-hover / active | #4096ff / #0958d9 | colorPrimaryHover / colorPrimaryActive |
| colors.surface-layout | #f5f5f5 | colorBgLayout |
| colors.surface | #ffffff | colorBgContainer |
| colors.surface-sider | #001529 | Layout Sider 深色主题 |
| colors.surface-table-header | #fafafa | Table.headerBg |
| colors.border / border-strong | #f0f0f0 / #d9d9d9 | colorBorderSecondary / colorBorder |
| colors.on-surface(-secondary/-tertiary/-quaternary) | 黑 88/65/45/25% | colorText / colorTextSecondary / colorTextTertiary / colorTextQuaternary |
| colors.success / warning / error | #52c41a / #faad14 / #ff4d4f | colorSuccess / colorWarning / colorError |
| rounded.sm / md / lg | 4px / 6px / 8px | tokens.radiusSm / radiusMd / borderRadius / borderRadiusLG |
| spacing.control-height | 32px | controlHeight |
| typography.body.fontSize | 14px | tokens.fontSizeBody / fontSize |
| typography.small.fontSize | 13px | tokens.fontSizeSmall |
| typography.tag.fontSize | 12px | tokens.fontSizeTag |
