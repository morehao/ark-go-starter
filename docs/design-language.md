# 已退役：前端设计语言

> ⚠️ **本文件已退役（Deprecated）。** 项目前端视觉规范已切换为 **Ant Design 原生风格**，事实源统一迁移至 [`docs/DESIGN.md`](./DESIGN.md)。

## 变更说明

- **旧方案**：「冷白工程台」（靛蓝 `#4f6ef7`、白侧栏、hairline 无阴影）。本文件是它的规范全文。
- **新方案**：Ant Design 原生（蓝色主色 `#1677ff`、深色 Sider `#001529`、6px 控件圆角 / 8px 卡片圆角），遵循 Google DESIGN.md 官方格式规范。
- **新事实源**：视觉规范 → [`docs/DESIGN.md`](./DESIGN.md)；代码事实源 → `packages/ui/src/theme.ts` 的 `tokens`。二者一一对应、人工同步。

## 迁移对照

| 旧（design-language.md） | 新（DESIGN.md） |
|---|---|
| 主色 靛蓝 `#4f6ef7` | 主色 蓝 `#1677ff`（白字承载用 `#0b63d6`） |
| 白侧栏 `#ffffff` | 深色 Sider `#001529` |
| 控件圆角 8px / 卡片 12px | 控件圆角 6px / 卡片 8px |
| 文字三级 hex 灰阶 | antd 透明度文字（88/65/45/25%） |
| 渐变品牌区 | AntD 蓝系（渐变仅作点缀） |

本文件仅作为历史记录与旧链接落点保留，**不再作为规范依据**。新增/修改样式请以 [`docs/DESIGN.md`](./DESIGN.md) 为准。
