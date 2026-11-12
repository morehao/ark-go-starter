// Package dtoproduct 定义商品管理接口的请求与响应 DTO。
package dtoproduct

import (
	"github.com/morehao/ark-go-starter/apps/api/object/objproduct"
	"github.com/morehao/ark-go-starter/pkg/operator"
	"github.com/morehao/golib/biz/gobject"
)

// ProductCreateReq 是创建商品的请求。
type ProductCreateReq struct {
	objproduct.ProductBaseInfo
	operator.Operator
}

// ProductCreateInTxReq 事务创建商品请求：在同一个事务里创建两条商品记录。
//
// FailAfterFirst 用于演示原子性：为 true 时，第一条插入成功后主动返回错误，
// 触发事务回滚，最终两条记录都不应落库。
type ProductCreateInTxReq struct {
	FirstName      string `json:"firstName" binding:"required"`  // 第一条商品名称（必填）
	SecondName     string `json:"secondName" binding:"required"` // 第二条商品名称（必填）
	FailAfterFirst bool   `json:"failAfterFirst"`                // 是否在第一条后制造错误以演示回滚
}

// ProductUpdateReq 修改商品管理请求。
//
// 业务字段用**指针**表达「是否传了该字段」：
//   - 指针为 nil 表示请求未携带该字段，service 侧不更新该列；
//   - 指针非 nil 表示显式赋值（含零值），service 侧按值更新。
//
// 这样既避免「客户端漏传字段把列清空」，也避免「传 0 被当作未传而无法清空」。
type ProductUpdateReq struct {
	ProductID  uint     `json:"-" uri:"productID" binding:"required"` // 自增 ID
	Name       *string  `json:"name" form:"name"`                     // 商品名称（nil 表示不修改）
	CategoryID *uint    `json:"categoryID" form:"categoryID"`         // 分类id（nil 表示不修改）
	Price      *float64 `json:"price" form:"price"`                   // 价格（nil 表示不修改）
	operator.Operator
}

// ProductDetailReq 是查询商品详情的请求。
type ProductDetailReq struct {
	ProductID uint `json:"-" uri:"productID" binding:"required"` // 自增 ID
}

// ProductPageListReq 商品管理列表查询请求。
//
// 过滤字段与 dao.ProductCond 一一对应，service 侧逐项映射；空值表示不按该条件过滤。
type ProductPageListReq struct {
	gobject.PageQuery
	Name       string `json:"name" form:"name"`             // 商品名称（可选，精确匹配）
	CategoryID uint   `json:"categoryID" form:"categoryID"` // 分类id（可选）
}

// ProductDeleteReq 是删除商品的请求。
type ProductDeleteReq struct {
	ProductID uint `json:"-" uri:"productID" binding:"required"` // 自增 ID
	operator.Operator
}
