// Package demo 是示例应用：装配 Gin 引擎、中间件、路由与基础设施。
package demo

import (
	"context"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo/config"

	// 注册 swag 生成的路由文档：gindocs.Register 依赖其 init 向 swag 注册文档。
	_ "github.com/morehao/ark-go-starter/demo/docs"
	"github.com/morehao/ark-go-starter/demo/internal/middleware"
	"github.com/morehao/ark-go-starter/demo/internal/router"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/ark-go-starter/pkg/iamlink"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/filestore"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/storage"

	// 注册 local 存储驱动：storage.New 按 driver 名 "local" 解析，必须 blank import 触发 init。
	_ "github.com/morehao/golib/storage/driver/local"
)

// AppName 是 demo 应用名，同时作为路由前缀与日志 service 标识。
const AppName = "demo"

// Routers 装配 Gin 引擎：注册中间件、路由与文件存储。
//
// 返回内嵌 IAM（Enable=true）的运行时 Closer 供 cmd/main.go defer 关闭；未启用时为 nil，
// 调用方需对返回值做 nil 保护（对 nil 调用 Close 已被 *iamRuntime 的 Close 方法做成安全）。
func Routers(engine *gin.Engine) io.Closer {
	// 先启动可选内嵌 IAM：Enable=false 时零副作用返回 nil；true 时挂载 /iam/* 路由并建表。
	iamCloser, err := SetupIAM(engine)
	if err != nil {
		panic(fmt.Errorf("demo.Routers: setup iam failed: %w", err))
	}

	middlewares := []gin.HandlerFunc{
		middleware.Example(),
	}
	// 鉴权中间件装配：
	//   - IAM 启用：业务路由挂 OIDC 本地验签中间件（真鉴权），身份经映射回填 uint；
	//   - IAM 关闭：仅 dev 挂演示身份中间件（信任请求头，生产禁用），生产拒绝。
	authMiddleware := buildAuthMiddleware()
	if authMiddleware != nil {
		middlewares = append(middlewares, authMiddleware)
	}

	routerGroups := ginserver.NewRouterGroups(engine, AppName, []ginserver.VersionGroup{
		{
			Version:     ginserver.ApiVersionV1,
			Middlewares: middlewares,
		},
	})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	if err := initFileStore(); err != nil {
		panic(fmt.Errorf("demo.Routers: init file store failed: %w", err))
	}

	router.RegisterRouter(routerGroups, AppName)

	return iamCloser
}

// buildAuthMiddleware 按 iam.mode 装配业务路由的鉴权中间件。
//
//   - op / rp：返回 OIDC 本地验签中间件（白名单 health/docs；/iam/* 不在业务分组内天然放行）。
//     rp 形态额外校验 token 的 aud（配置了 iam.audience 时），防止接受其它应用的 token；
//     op 形态为同进程内嵌，无需 aud 校验。
//   - disabled：dev 返回演示身份中间件；非 dev 返回 nil（无身份，业务按未登录处理）。
func buildAuthMiddleware() gin.HandlerFunc {
	mode := config.Conf.IAM.EffectiveMode()
	if mode == config.ModeOP || mode == config.ModeRP {
		rt := CurrentIAM()
		if rt == nil {
			// SetupIAM 已成功但运行时缺失属装配错误，fail-closed：拒绝而非放行。
			panic("demo.buildAuthMiddleware: iam enabled but runtime is nil")
		}
		// 业务白名单：健康检查与 swag 文档免鉴权。业务路由统一在 /v1/demo 下。
		opts := []middleware.OIDCAuthOption{
			middleware.WithOIDCKeySource(rt.keySource),
			middleware.WithOIDCIssuer(rt.issuer),
			middleware.WithLocalUserMapper(iamlink.NewSvc()),
			middleware.WithOIDCSkipPaths("/"+AppName+"/docs", "/v1/"+AppName+"/health"),
		}
		// rp 形态才需 aud 校验（内嵌 op 同进程无需）；aud 为空则不追加。
		if mode == config.ModeRP && config.Conf.IAM.Audience != "" {
			opts = append(opts, middleware.WithOIDCAudiences(config.Conf.IAM.Audience))
		}
		return middleware.OIDCAuth(opts...)
	}

	// 演示用身份中间件：只有 dev 环境挂载，让 controller 能取到操作人；
	// 它信任客户端请求头，生产环境绝不能启用（详见 middleware.DemoIdentity 注释）。
	if config.Conf.Server.Env == "dev" {
		return middleware.DemoIdentity()
	}
	return nil
}

// initFileStore 根据配置初始化文件上传存储（storage 驱动 + filestore 单例）。
func initFileStore() error {
	cfg := config.Conf.FileStorage

	st, err := storage.New(cfg.Driver, cfg.Storage)
	if err != nil {
		return fmt.Errorf("storage.New: %w", err)
	}

	var storeOpts []filestore.StoreOption
	if cfg.Storage.SignSecret != "" {
		storeOpts = append(storeOpts, filestore.WithSignSecret(cfg.Storage.SignSecret))
	}

	filestore.Init(dbclient.DemoDB(context.TODO()), st, cfg.Bucket, storeOpts...)
	glog.Infof(context.TODO(), "[demo.initFileStore] filestore init done, driver=%s, bucket=%s, local=%v", cfg.Driver, cfg.Bucket, filestore.IsLocal())
	return nil
}
