[English](./README.md) | [简体中文](./README.zh.md)

# 快速开始

clone 之后**一条命令**把前后端跑起来（仅 dev）：

```bash
# 有 Docker（推荐）：自动起 PostgreSQL + Redis，建库建表、初始化 IAM，再起后端(:8099) 与前端(:3000)
make quickstart

# 无 Docker：复用本机已在运行的 PostgreSQL(5432) / Redis(6379)
make quickstart-local
```

然后打开 **http://localhost:3000**，会自动跳到登录页；用命令终端打印的账号密码登录
（dev 缺省：`admin` / `Adm1n@Strong!Pass`，可用 `IAM_ADMIN_USER` / `IAM_ADMIN_PASS` 覆盖）。
命令幂等，可重复执行不报错、不重置数据。

验证全流程（登录 → 商品 CRUD → 字典 → 文件上传 → 异步 → 定时）：

```bash
make quickstart-verify IAM_ADMIN_USER=admin IAM_ADMIN_PASS='你的密码'
```

> 完整说明（操作清单、常见坑、手动分步、安全须知）见
> [docs/agents/quickstart.md](./docs/agents/quickstart.md)。

---

# 项目简介
`ark-go-starter` 是一个前后端一体的 Go Web 全栈工程实践项目：后端基于 [Gin](https://github.com/gin-gonic/gin) + GORM（Go workspace 多模块），前端基于 React + Vite + Ant Design（pnpm monorepo），提供分层清晰、可维护、可扩展的多应用服务结构。
# 项目特点

- 清晰的项目结构：参考了[project-layout](https://github.com/golang-standards/project-layout)，遵循分层架构思想，采用 monorepo 布局：`apps/`（应用：`apps/api` Go 后端、`apps/web` React 前端）、`packages/`（前端共享包）、`pkg/`（Go 公共层）、`tests/e2e`、`scripts/`、`deploy/`，目录组织合理，便于团队协作与长期维护。
- 常用组件集成：PostgreSQL、Redis（已移除 Elasticsearch 集成，PostgreSQL 为唯一目标库）。
- 全链路日志追踪：基于`zap`封装的日志组件`glog`,支持链路 ID 贯穿 PostgreSQL、Redis、Http 调用。
- 代码生成工具：提供`gocli`命令行工具，支持根据配置快速生成标准代码（包括 model、dao、object、dto、code、service、controller、router层代码）。
- `Swagger`文档支持：使用`swaggo`自动生成 API 文档，方便前后端联调与测试。
- 前端 Monorepo：基于 pnpm workspace 共享 `packages/`（tsconfig/types/api/ui），业务应用放在 `apps/web`。
- Docker 支持： 提供基础的`Dockerfile`，实现容器化部署
- 丰富的`Makefile`工具链：支持`make`命令快速构建、运行、代码生成、接口文档生成、docker 部署、前端启动等基础操作。
- 逐渐丰富的`golib`库：对常用组件封装，使用更友好。
- 可选账号认证（ark-iam 三态）：由 `iam.mode` 决定——`disabled`（不接入）、`op`（本进程内嵌 OIDC Provider，本项目自身就是 IAM）、`rp`（对接外部 ark-iam，本地验签）。**仓库自带的 dev 配置（`apps/api/config/config.yaml`）为 `op`，即登录强制**（quickstart 会打印管理员账号密码）；自带 prod 配置为 `disabled`，`mode` 留空时也回落为 `disabled`。详见 [docs/agents/iam.md](./docs/agents/iam.md)。

# 项目结构

参考了[project-layout](https://github.com/golang-standards/project-layout)。当前项目结构如下：
```bash
.
├── apps
│   ├── api                     # Go 后端应用（Gin，:8099；逻辑名 demo → 目录 api）
│   │   ├── cmd                 # 入口函数
│   │   ├── client              # 外部客户端
│   │   ├── config              # 配置
│   │   ├── dao                 # 数据访问层
│   │   ├── model               # 数据模型
│   │   ├── docs                # Swagger 文档
│   │   ├── e2e                 # 后端真端口 e2e
│   │   ├── internal            # controller / dto / router / service / middleware
│   │   └── object              # 基础对象
│   └── web                     # React 前端应用（Vite + React + AntD，:3000）
├── packages                    # 前端共享包（api / api-client / types / ui / tsconfig）
├── pkg                         # Go 公共层（code / dbclient / testsetup / token / ginhelper / iamlink / operator）
├── tests
│   └── e2e                     # 端到端测试（Playwright specs + A4 golden）
├── scripts                     # 仓库级脚本（quickstart.sh）
├── deploy                      # 容器与编排（docker/api.Dockerfile、compose/dev.yml）
├── contracts                   # 契约平面（OpenAPI 3.1 契约源 + verify.sh）
├── docs                        # 文档（agents/ 规范、DESIGN.md 视觉规范）
├── go.work                     # 根 Go workspace
├── package.json                # 根前端 workspace 清单
├── pnpm-workspace.yaml         # 根 pnpm workspace 定义
├── Makefile                    # 根 Makefile（后端 + 前端命令）
└── AGENTS.md                   # AI 代理入口（开发规范索引）
```

# 基础功能

## 代码生成

安装命令行终端
```bash
go install github.com/morehao/gocli@latest
```
确保项目应用目录下有代码生成配置文件，示例：`apps/api/config/code_gen.yaml`。代码生成命令如下：
```bash
# 基于表生成整个功能模块
make codegen APP=demo COMMAND=module
# 生成model代码
make codegen APP=demo COMMAND=model
# 生成单个接口代码
make codegen APP=demo COMMAND=api
```
代码生成详细说明文档见[generate](https://github.com/morehao/gocli?tab=readme-ov-file#generate)。

## 接口文档

安装swag工具
```shell
go install github.com/swaggo/swag/cmd/swag@latest
```
生成接口文档
```shell
make swag APP=demo
```
访问接口文档
开发环境访问 `http://localhost:8099/demo/redocs` 即可查看接口文档。

## 前端开发

在**仓库根**（pnpm workspace 根）安装依赖并启动：
```bash
pnpm install
pnpm dev        # 默认 :3000，代理 /v1 → http://localhost:8099
```
也可通过 `make dev-frontend` 启动、`make stop-frontend` 停止。
> 前端依赖后端服务，请先执行 `make run APP=demo` 启动后端。

## 项目部署
构建镜像
```bash
make docker-build APP=demo
```
运行容器
```bash
make docker-run APP=demo
```

## 快速生成新项目
安装`cutter`
```shell
go install github.com/morehao/gocli@latest
```
在 **当前项目根目录下（即`./`）** 执行命令
```shell
gocli cutter -d /goProject/yourAppName
```
执行后，会以当前项目为模板项目，在`/goProject`目录下生成一个名为`yourAppName`的项目。

`go-cutter`是一个快速生成项目代码的命令行工具，可以基于现有项目快速生成一个新的项目，具体使用方法请参考 [cutter](https://github.com/morehao/gocli)。


## 相关组件
相关组件均在[golib](https://github.com/morehao/golib)包中实现。
