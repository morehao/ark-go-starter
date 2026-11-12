package dtoproduct

import (
	"github.com/morehao/ark-go-starter/apps/api/object/objproduct"
	"github.com/morehao/golib/biz/gobject"
)

// ProductCreateResp 是创建商品的响应。
type ProductCreateResp struct {
	ProductID uint `json:"productID"` // 自增 ID
}

// ProductCreateInTxResp 事务创建商品响应。
type ProductCreateInTxResp struct {
	ProductIDs []uint `json:"productIDs"` // 事务内创建成功的商品 ID 列表
}

// ProductDetailResp 是商品详情的响应。
type ProductDetailResp struct {
	ProductID uint `json:"productID"` // 自增 ID
	objproduct.ProductBaseInfo
	gobject.OperatorBaseInfo
}

// ProductPageListItem 是商品分页列表的单项。
type ProductPageListItem struct {
	ProductID uint `json:"productID"` // 自增 ID
	objproduct.ProductBaseInfo
	gobject.OperatorBaseInfo
}

// ProductPageListResp 是商品分页列表的响应。
type ProductPageListResp struct {
	List  []ProductPageListItem `json:"list"`  // 数据列表
	Total int64                 `json:"total"` // 数据总条数
}
