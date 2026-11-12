// Package ctrproduct 是商品管理控制器：绑定请求、填充操作人并调用 service。
package ctrproduct

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/api/internal/dto/dtoproduct"
	"github.com/morehao/ark-go-starter/apps/api/internal/service/svcproduct"
	"github.com/morehao/ark-go-starter/libs/go/ginhelper"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// ProductCtr 是商品管理控制器接口。
type ProductCtr interface {
	Create(ctx *gin.Context)
	CreateInTx(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type productCtr struct {
	productSvc svcproduct.ProductSvc
}

var _ ProductCtr = (*productCtr)(nil)

// NewProductCtr 构造 ProductCtr。
func NewProductCtr() ProductCtr {
	return &productCtr{
		productSvc: svcproduct.NewProductSvc(),
	}
}

// Create 创建商品管理
// @Tags 商品管理
// @Summary 创建商品管理
// @accept application/json
// @Produce application/json
// @Param req body dtoproduct.ProductCreateReq true "创建商品管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoproduct.ProductCreateResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/demo/products [post].
func (ctr *productCtr) Create(ctx *gin.Context) {
	var req dtoproduct.ProductCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	ginhelper.FillOperator(ctx, &req.Operator)

	res, err := ctr.productSvc.Create(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// CreateInTx 事务示例：同一事务内创建两条商品记录
// @Tags 商品管理
// @Summary 事务示例：同一事务内创建两条商品记录
// @accept application/json
// @Produce application/json
// @Param req body dtoproduct.ProductCreateInTxReq true "事务创建请求"
// @Success 200 {object} gincontext.DtoRender{data=dtoproduct.ProductCreateInTxResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/demo/products/tx [post].
func (ctr *productCtr) CreateInTx(ctx *gin.Context) {
	var req dtoproduct.ProductCreateInTxReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.productSvc.CreateInTx(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// Delete 删除商品管理
// @Tags 商品管理
// @Summary 删除商品管理
// @accept application/json
// @Produce application/json
// @Param productID path int true "productID"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/demo/products/{productID} [delete].
func (ctr *productCtr) Delete(ctx *gin.Context) {
	var req dtoproduct.ProductDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	ginhelper.FillOperator(ctx, &req.Operator)

	if err := ctr.productSvc.Delete(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// Update 修改商品管理
// @Tags 商品管理
// @Summary 修改商品管理
// @accept application/json
// @Produce application/json
// @Param productID path int true "productID"
// @Param req body dtoproduct.ProductUpdateReq true "修改商品管理"
// @Success 200 {object} gincontext.DtoRender{data=string} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "修改成功"}"
// @Router /v1/demo/products/{productID} [put].
func (ctr *productCtr) Update(ctx *gin.Context) {
	var req dtoproduct.ProductUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	ginhelper.FillOperator(ctx, &req.Operator)

	if err := ctr.productSvc.Update(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// Detail 商品管理详情
// @Tags 商品管理
// @Summary 商品管理详情
// @accept application/json
// @Produce application/json
// @Param productID path int true "productID"
// @Success 200 {object} gincontext.DtoRender{data=dtoproduct.ProductDetailResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/demo/products/{productID} [get].
func (ctr *productCtr) Detail(ctx *gin.Context) {
	var req dtoproduct.ProductDetailReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.productSvc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// PageList 商品管理列表
// @Tags 商品管理
// @Summary 商品管理列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtoproduct.ProductPageListReq true "商品管理列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoproduct.ProductPageListResp} "{"code": 0, "requestID": "xxx", "data": "ok", "msg": "success"}"
// @Router /v1/demo/products [get].
func (ctr *productCtr) PageList(ctx *gin.Context) {
	var req dtoproduct.ProductPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.productSvc.PageList(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
