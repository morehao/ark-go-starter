# API 路由与 Swagger 规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## API 路由规范

- **路径格式**: RESTful 资源式，`/{版本}/{app}/{资源复数}` + HTTP 方法，如 `GET /v1/demo/users`
- **版本号**: 放在路径最前，使用 `/v1/`, `/v2/` 等格式
- **App 标识**: 用于区分不同应用，如 `demo`
- **资源名**: 复数形式，小写 kebab-case（如 `user` → `users`、`user-info`）；路径参数用 `:xxxID`（如 `:userID`）
- **HTTP 方法**: 创建 `POST`、列表 `GET`、详情 `GET /:id`、修改 `PUT /:id`、删除 `DELETE /:id`

**路由层级限制为 4 层**，避免使用多层嵌套路径。

### 路由示例

| 模块 | 操作 | 完整路径 |
|------|------|----------|
| user | 创建 | `POST /v1/demo/users` |
| user | 列表 | `GET /v1/demo/users` |
| user | 详情 | `GET /v1/demo/users/:userID` |
| user | 修改 | `PUT /v1/demo/users/:userID` |
| user | 删除 | `DELETE /v1/demo/users/:userID` |

### 路由注册

在 `internal/router/*.go` 中注册路由，先按版本分组，再按资源分组：

```go
func userRouter(groups *ginserver.RouterGroups) {
    userCtr := ctruser.NewUserCtr()
    v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

    v1RouterGroup.POST("/users", userCtr.Create)
    v1RouterGroup.GET("/users", userCtr.PageList)
    v1RouterGroup.GET("/users/:userID", userCtr.Detail)
    v1RouterGroup.PUT("/users/:userID", userCtr.Update)
    v1RouterGroup.DELETE("/users/:userID", userCtr.Delete)
}
```

## Swagger 文档

使用 Swag Go 注解，需包含以下注释：

```go
// @Tags 用户管理
// @Summary 创建用户管理
// @accept application/json
// @Produce application/json
// @Param req body dtouser.UserCreateReq true "创建用户管理"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.UserCreateResp}
// @Router /v1/demo/user/create [post]
```

生成文档：

```bash
make swag APP=demo
```

## 相关文档

- Controller / Service 分层与返回模式：[backend-conventions.md](./backend-conventions.md)
- 命令大全：[commands.md](./commands.md)
