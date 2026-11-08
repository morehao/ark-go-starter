// Package router 负责注册各业务路由到 Gin 路由分组。
package router

import (
	"context"

	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/golib/biz/gserver/ginasync"
	"github.com/morehao/golib/biz/gserver/gincron"
	"github.com/morehao/golib/biz/gserver/gindict"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/biz/gserver/ginsetting"
	"github.com/morehao/golib/biz/gserver/ginupload"
	"github.com/morehao/golib/filestore"
	"github.com/morehao/golib/glog"
)

// RegisterRouter 注册全部业务路由到 v1 分组。
func RegisterRouter(groups *ginserver.RouterGroups, _ string) {
	formatRouter(groups)
	sseRouter(groups)
	clientRouter(groups)
	healthRouter(groups)
	fileRouter(groups)
	productRouter(groups)
	registerGolibModules(groups)
}

// registerGolibModules 注册 golib 自带的字典/设置/定时任务/异步任务路由模块到 v1 分组。
// 组件实例为 nil 时跳过并告警，避免 panic。
func registerGolibModules(groups *ginserver.RouterGroups) {
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

	if instance := config.GetDict(); instance != nil {
		gindict.Register(v1RouterGroup, instance)
	} else {
		glog.Warnf(context.Background(), "[%s.registerGolibModules] dict instance is nil, skip gindict register", "router")
	}

	if instance := config.GetSetting(); instance != nil {
		ginsetting.Register(v1RouterGroup, instance)
	} else {
		glog.Warnf(context.Background(), "[%s.registerGolibModules] setting instance is nil, skip ginsetting register", "router")
	}

	if instance := config.GetCron(); instance != nil {
		gincron.Register(v1RouterGroup, instance)
	} else {
		glog.Warnf(context.Background(), "[%s.registerGolibModules] cron instance is nil, skip gincron register", "router")
	}

	if instance := config.GetAsync(); instance != nil {
		ginasync.Register(v1RouterGroup, instance)
	} else {
		glog.Warnf(context.Background(), "[%s.registerGolibModules] async instance is nil, skip ginasync register", "router")
	}
}

// fileRouter 注册 golib ginupload 文件上传路由
func fileRouter(groups *ginserver.RouterGroups) {
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	ginupload.Register(v1RouterGroup, filestore.Get())
}
