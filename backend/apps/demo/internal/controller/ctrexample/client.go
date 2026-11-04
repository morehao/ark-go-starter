package ctrexample

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo/client/httpbingo"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

// ClientCtr 是外部 client 调用示例控制器接口。
type ClientCtr interface {
	CallGetHttpbingo(ctx *gin.Context)
}

type clientCtr struct{}

var _ ClientCtr = (*clientCtr)(nil)

// NewClientCtr 构造 ClientCtr。
func NewClientCtr() ClientCtr {
	return &clientCtr{}
}

func (ctr *clientCtr) CallGetHttpbingo(ctx *gin.Context) {
	getReq := httpbingo.GetRequest{
		ID: 1,
	}
	res, err := httpbingo.Get(ctx, &getReq)
	if err != nil {
		glog.Errorf(ctx, "[CallGetHttpbingo] get failed, err: %v", err)
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
