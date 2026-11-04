// Package router 负责注册各业务路由到 Gin 路由分组。
package router

import (
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/morehao/golib/biz/gserver/ginupload"
	"github.com/morehao/golib/filestore"
)

// RegisterRouter 注册全部业务路由到 v1 分组。
func RegisterRouter(groups *ginserver.RouterGroups, _ string) {
	formatRouter(groups)
	sseRouter(groups)
	clientRouter(groups)
	healthRouter(groups)
	fileRouter(groups)
	userRouter(groups)
}

// fileRouter 注册 golib ginupload 文件上传路由
func fileRouter(groups *ginserver.RouterGroups) {
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	ginupload.Register(v1RouterGroup, filestore.Get())
}
