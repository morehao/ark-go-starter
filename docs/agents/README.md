# ark-go-starter 开发规范文档中心

> 上级入口：[AGENTS.md](../../AGENTS.md)　（AI 代理总入口：项目概述 + 命令速查 + 硬性规则速查）

本目录承载从 `AGENTS.md` 拆分出来的详细开发规范。拆分的目的是让入口文件保持精简、规范正文按主题独立维护、便于按需加载。

## 文档索引

| 文档 | 内容 | 何时阅读 |
|------|------|----------|
| [project-overview.md](./project-overview.md) | 项目定位、多模块目录结构、应用与端口 | 初次接触代码库 |
| [commands.md](./commands.md) | 构建 / 运行 / 测试 / Lint / 代码生成 / Swagger / Docker 命令 | 需要执行命令时 |
| [backend-conventions.md](./backend-conventions.md) | 后端分层结构、命名、注释（DTO 字段 / SQL 字段 comment）、模型字段类型（PostgreSQL）、常量、Import 排序、接口与依赖注入、错误处理、事务、Controller 返回 | 编写后端代码前 |
| [frontend-conventions.md](./frontend-conventions.md) | 前端 monorepo 包结构、命令、`AppShell`/`PageContainer`/`tokens` 硬规则、新增页面检查清单 | 编写前端代码前 |
| [../frontend/DESIGN.md](../../frontend/DESIGN.md) | 前端设计语言：色彩/排版/间距/圆角/组件规格/Do's and Don'ts（视觉规范事实源） | 改样式、配色、布局前 |
| [api-conventions.md](./api-conventions.md) | 路由路径规范、路由注册、Swagger 注解 | 新增或修改接口时 |
| [testing.md](./testing.md) | 测试文件组织、测试环境初始化、断言写法 | 编写测试时 |
| [golib.md](./golib.md) | golib 公共组件库能力说明：项目内封装层、golib 组件速查（已用/可用）、驱动注册、内置表与升级约定 | 写新功能、引入通用组件前（能力复用） |
| [toolchain.md](./toolchain.md) | 依赖、API 文档、代码生成、存储、链路追踪、日志等工具选型 | 引入或替换组件时 |

## 阅读顺序建议

1. 先读入口 [AGENTS.md](../../AGENTS.md)：项目概述、命令速查、硬性规则速查；
2. 写后端代码 → [backend-conventions.md](./backend-conventions.md)；
3. 写前端代码 → [frontend-conventions.md](./frontend-conventions.md)，改样式再读 [`frontend/DESIGN.md`](../../frontend/DESIGN.md)；
4. 动接口 → [api-conventions.md](./api-conventions.md)；
5. 需要通用能力（日志 / HTTP / 锁 / 限流 / 任务 / Excel 等）→ [golib.md](./golib.md)；
6. 补测试 → [testing.md](./testing.md)。

## 维护约定

- 入口 [AGENTS.md](../../AGENTS.md) 只保留：项目概述、项目结构、命令速查、硬性规则速查（每条一句话 + 详情链接）；
- 扩写规范时优先修改本目录下已有主题文档，**不要**把细节堆回 `AGENTS.md`；
- 确实出现新主题时才在本目录新建文件，并同步更新上面的索引表与 `AGENTS.md` 的导航；
- 文档内链接一律使用仓库相对路径，保证在 GitHub / 编辑器中可直接跳转；
- 术语、路径、常量名发生变化时，先改本目录正文，再检查 `AGENTS.md` 速查条目是否同步。
