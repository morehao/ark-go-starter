// Package ctrhealth 是健康检查控制器。
package ctrhealth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/api/internal/service/svchealth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// HealthCtr 是健康检查控制器接口。
type HealthCtr interface {
	Check(ctx *gin.Context)
}

type healthCtr struct {
	healthSvc svchealth.HealthSvc
}

var _ HealthCtr = (*healthCtr)(nil)

// NewHealthCtr 构造 HealthCtr。
func NewHealthCtr() HealthCtr {
	return &healthCtr{
		healthSvc: svchealth.NewHealthSvc(),
	}
}

// Check 健康检查
// @Tags 健康检查
// @Summary 健康检查
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtohealth.DbCheckResp} "{"code": 0,"data": "ok","msg": "success"}"
// @Router /v1/demo/health [get].
func (ctr *healthCtr) Check(ctx *gin.Context) {
	gincontext.Success(ctx, ctr.healthSvc.Check(ctx.Request.Context()))
}
