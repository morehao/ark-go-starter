// Package main 是 demo 应用入口：初始化配置、日志、DB、Redis、链路后启动 HTTP 服务。
package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/golib/glog"
)

func main() {
	if err := serverInit(); err != nil {
		panic(fmt.Sprintf("server init failed, error: %v", err))
	}
	if config.Conf.Server.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	defer shutdownTraceProvider()
	defer func() {
		if err := glog.Close(); err != nil {
			fmt.Printf("failed to close logger: %v\n", err)
		}
	}()

	engine := gin.New()
	engine.Use(gin.Recovery())
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
