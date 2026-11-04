// Package demo 是示例应用：装配 Gin 引擎、中间件、路由与基础设施。
package demo

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo/config"

	// 注册 swag 生成的路由文档：gindocs.Register 依赖其 init 向 swag 注册文档。
	_ "github.com/morehao/ark-go-starter/demo/docs"
	"github.com/morehao/ark-go-starter/demo/internal/middleware"
	"github.com/morehao/ark-go-starter/demo/internal/router"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
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
func Routers(engine *gin.Engine) {
	middlewares := []gin.HandlerFunc{
		middleware.Example(),
	}
	// 演示用身份中间件：只有 dev 环境挂载，让 controller 能取到操作人；
	// 它信任客户端请求头，生产环境绝不能启用（详见 middleware.DemoIdentity 注释）。
	if config.Conf.Server.Env == "dev" {
		middlewares = append(middlewares, middleware.DemoIdentity())
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
