[English](./README.md) | [简体中文](./README.zh.md)

# 项目简介
`ark-go-starter` 是一个前后端一体的 Go Web 全栈工程实践项目：后端基于 [Gin](https://github.com/gin-gonic/gin) + GORM（Go workspace 多模块），前端基于 React + Vite + Ant Design（pnpm monorepo），提供分层清晰、可维护、可扩展的多应用服务结构。
# 项目特点

- 清晰的项目结构：参考了[project-layout](https://github.com/golang-standards/project-layout)，遵循分层架构思想，前后端分离（`backend/` + `frontend/`），目录组织合理，便于团队协作与长期维护。
- 常用组件集成：MySQL、Redis、ES。
- 全链路日志追踪：基于`zap`封装的日志组件`glog`,支持链路 ID 贯穿 MySQL、Redis、ES、Http 调用。
- 代码生成工具：提供`gocli`命令行工具，支持根据配置快速生成标准代码（包括 model、dao、object、dto、code、service、controller、router层代码）。
- `Swagger`文档支持：使用`swaggo`自动生成 API 文档，方便前后端联调与测试。
- 前端 Monorepo：基于 pnpm workspace 共享 `packages/`（tsconfig/types/api），业务应用放在 `apps/`。
- Docker 支持： 提供基础的`Dockerfile`，实现容器化部署
- 丰富的`Makefile`工具链：支持`make`命令快速构建、运行、代码生成、接口文档生成、docker 部署、前端启动等基础操作。
- 逐渐丰富的`golib`库：对常用组件封装，使用更友好。

# 项目结构

参考了[project-layout](https://github.com/golang-standards/project-layout)。当前项目结构如下：
```bash
.
├── backend                 # Go 后端项目（go.work 多模块）
│   ├── apps
│   │   └── demo            # Demo 示例应用
│   │       ├── cmd         # 入口函数
│   │       ├── client      # 外部客户端
│   │       ├── config      # 配置
│   │       ├── dao         # 数据访问层
│   │       ├── model       # 数据模型
│   │       ├── docs        # Swagger 文档
│   │       ├── internal    # controller / dto / router / service / middleware
│   │       ├── object      # 基础对象
│   │       └── scripts     # Dockerfile 等
│   ├── pkg                 # 公共包（code / dbclient / testsetup / token）
│   ├── go.work             # Go workspace 管理文件
│   └── output              # 构建产物
├── frontend                # React 前端项目（pnpm monorepo）
│   ├── apps
│   │   └── demo-web        # 演示前端应用（Vite + React + AntD）
│   └── packages            # 共享包（tsconfig / types / api）
├── Makefile                # 根 Makefile（backend + frontend 双端命令）
├── AGENTS.md               # 开发规范
└── docs                    # 文档
```

# 基础功能

## 代码生成

安装命令行终端
```bash
go install github.com/morehao/gocli@latest
```
确保项目应用目录下有代码生成配置文件，示例：`ark-go-starter/apps/demo/config/code_gen.yaml`。代码生成命令如下：
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

进入 `frontend/` 目录安装依赖并启动：
```bash
cd frontend
pnpm install
pnpm dev        # 默认 :3000，代理 /v1 → http://localhost:8099
```
也可在项目根目录通过 `make dev-frontend` 启动、`make stop-frontend` 停止。
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
