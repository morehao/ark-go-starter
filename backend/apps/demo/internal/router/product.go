package router

import (
	"github.com/morehao/ark-go-starter/demo/internal/controller/ctrproduct"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// productRouter 初始化商品管理路由信息
func productRouter(groups *ginserver.RouterGroups) {
	productCtr := ctrproduct.NewProductCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

	v1RouterGroup.POST("/products", productCtr.Create)
	v1RouterGroup.POST("/products/tx", productCtr.CreateInTx)
	v1RouterGroup.GET("/products", productCtr.PageList)
	v1RouterGroup.GET("/products/:productID", productCtr.Detail)
	v1RouterGroup.PUT("/products/:productID", productCtr.Update)
	v1RouterGroup.DELETE("/products/:productID", productCtr.Delete)
}
