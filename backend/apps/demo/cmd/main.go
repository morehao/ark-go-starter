// Package main 是 demo 应用入口：初始化配置、日志、DB、Redis、链路后启动 HTTP 服务。
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-go-starter/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/glog"
)

func main() {
	if err := serverInit(); err != nil {
		panic(fmt.Sprintf("server init failed, error: %v", err))
	}
	if config.Conf.Server.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	// 关闭顺序（defer 为 LIFO）：先停调度器/异步 worker（它们退出时仍会写日志），
	// 再关日志、最后关 trace provider；顺序错了会出现"向已关闭的 logger 写入"并 panic。
	defer shutdownTraceProvider()
	defer func() {
		if err := glog.Close(); err != nil {
			fmt.Printf("failed to close logger: %v\n", err)
		}
	}()
	defer shutdownComponents()

	engine := gin.New()
	// 统一错误出口：
	//   1) panic 也要回 JSON envelope（gin 默认 Recovery 返回 500 空 body，客户端无法统一解析）；
	//   2) 未命中路由返回 JSON 404，而不是 text/plain "404 page not found"。
	engine.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		glog.Errorf(context.Background(), "[%s] panic recovered, method:%s path:%s panic:%v",
			demo.AppName, c.Request.Method, c.Request.URL.Path, recovered)
		gincontext.FailWithStatus(c, http.StatusInternalServerError, code.GetError(gconstant.SystemErrorErr))
	}))
	engine.NoRoute(func(c *gin.Context) {
		gincontext.FailWithStatus(c, http.StatusNotFound, code.GetError(code.RouteNotFoundError))
	})
	// 默认 gin 信任任意代理，client.address 直接取 X-Forwarded-For 的客户端可控值：
	// 审计日志、按 IP 限流/封禁都能被一个请求头绕过（启动日志里的 "You trusted all proxies" 警告）。
	// 这里显式关闭信任（ClientIP 回落为 TCP 对端地址）。若部署在自有网关/负载均衡之后，
	// 应改为 SetTrustedProxies([]string{"<代理网段>"})，只信任该网段转发来的 XFF。
	if err := engine.SetTrustedProxies(nil); err != nil {
		glog.Errorf(context.Background(), "%s set trusted proxies fail, err:%v", demo.AppName, err)
	}
	// Routers 返回可选内嵌 IAM 的运行时 Closer（未启用时为 nil，Close 为安全空操作）。
	if iamCloser := demo.Routers(engine); iamCloser != nil {
		defer func() {
			if err := iamCloser.Close(); err != nil {
				glog.Errorf(context.Background(), "%s close iam fail, err:%v", demo.AppName, err)
			}
		}()
	}

	if err := engine.Run(":" + config.Conf.Server.Port); err != nil {
		glog.Errorf(context.Background(), "%s run fail, port:%s", demo.AppName, config.Conf.Server.Port)
		panic(err)
	}
	glog.Infof(context.Background(), "%s run success, port:%s", demo.AppName, config.Conf.Server.Port)
}
