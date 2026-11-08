# API 路由与 Swagger 规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## API 路由规范

- **路径格式**: RESTful 资源式，`/{版本}/{app}/{资源复数}` + HTTP 方法，如 `GET /v1/demo/products`
- **版本号**: 放在路径最前，使用 `/v1/`, `/v2/` 等格式
- **App 标识**: 用于区分不同应用，如 `demo`
- **资源名**: 复数形式，小写 kebab-case（如 `product` → `products`、`user-info`）；路径参数用 `:xxxID`（如 `:productID`）
- **HTTP 方法**: 创建 `POST`、列表 `GET`、详情 `GET /:id`、修改 `PUT /:id`、删除 `DELETE /:id`

**路由层级限制为 4 层**，避免使用多层嵌套路径。

### 路由示例

| 模块 | 操作 | 完整路径 |
|------|------|----------|
| product | 创建 | `POST /v1/demo/products` |
| product | 列表 | `GET /v1/demo/products` |
| product | 详情 | `GET /v1/demo/products/:productID` |
| product | 修改 | `PUT /v1/demo/products/:productID` |
| product | 删除 | `DELETE /v1/demo/products/:productID` |

### 路由注册

在 `internal/router/*.go` 中注册路由，先按版本分组，再按资源分组：

```go
func productRouter(groups *ginserver.RouterGroups) {
    productCtr := ctrproduct.NewProductCtr()
    v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

    v1RouterGroup.POST("/products", productCtr.Create)
    v1RouterGroup.GET("/products", productCtr.PageList)
    v1RouterGroup.GET("/products/:productID", productCtr.Detail)
    v1RouterGroup.PUT("/products/:productID", productCtr.Update)
    v1RouterGroup.DELETE("/products/:productID", productCtr.Delete)
}
```

### golib 自带路由模块

除业务 `router.go` 外，golib 的 `gindict`（数据字典）、`ginsetting`（配置中心）、`gincron`（定时任务）、`ginasync`（异步任务）自带 RESTful 资源式路由。它们是 `biz/gserver/*` 下的 `Register` 风格模块，签名统一为 `Register(group *gin.RouterGroup, dep)`，取出 v1 分组后把对应组件实例传进去即可挂载：

```go
func registerGolibModules(groups *ginserver.RouterGroups) {
    v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

    gindict.Register(v1RouterGroup, config.GetDict())       // 依赖 dict.Dict
    ginsetting.Register(v1RouterGroup, config.GetSetting()) // 依赖 gsetting.Setting
    gincron.Register(v1RouterGroup, config.GetCron())       // 依赖 gcron.Scheduler
    ginasync.Register(v1RouterGroup, config.GetAsync())     // 依赖 gasync.Server
}
```

- 组件实例在启动期初始化（demo 用主库 PostgreSQL，组件自带表默认 `AutoMigrate`），路由层只负责 `Register`；实例为 `nil`（如某环境关闭 async）时应跳过注册并告警，避免 panic。
- 挂载后路由同样位于 `/{版本}/{app}/...` 下，如 `POST /v1/demo/dict-types`、`GET /v1/demo/settings`、`GET /v1/demo/cron-tasks`、`GET /v1/demo/async-tasks`。能力与接入边界见 [golib.md](./golib.md)。

## Swagger 文档

使用 Swag Go 注解，需包含以下注释：

```go
// @Tags 商品管理
// @Summary 创建商品管理
// @accept application/json
// @Produce application/json
// @Param req body dtoproduct.ProductCreateReq true "创建商品管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoproduct.ProductCreateResp}
// @Router /v1/demo/products [post]
```

生成文档：

```bash
make swag APP=demo
```

## 相关文档

- Controller / Service 分层与返回模式：[backend-conventions.md](./backend-conventions.md)
- 命令大全：[commands.md](./commands.md)
