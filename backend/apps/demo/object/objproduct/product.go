// Package objproduct 定义商品相关的领域对象，供 DTO 组合复用。
package objproduct

// ProductBaseInfo 商品基础信息（创建、详情、列表共用）。
type ProductBaseInfo struct {
	Name       string  `json:"name" form:"name"`             // 商品名称
	CategoryID uint    `json:"categoryID" form:"categoryID"` // 分类id
	Price      float64 `json:"price" form:"price"`           // 价格
}
