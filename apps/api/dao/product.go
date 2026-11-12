// Package dao 是数据访问层：封装 GORM 查询条件与 DAO 构造。
package dao

import (
	"github.com/morehao/ark-go-starter/apps/api/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProductCond 是商品列表的查询条件。
type ProductCond struct {
	*gormdao.BaseCond
	Name       string
	CategoryID uint
	Price      float64
	CreatedBy  uint
	UpdatedBy  uint
	DeletedBy  uint
}

// BuildCondition 把非零字段拼成 GORM 查询条件。
func (c *ProductCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "name"}, Value: c.Name})
	}
	if c.CategoryID != 0 {
		db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "category_id"}, Value: c.CategoryID})
	}
	if c.Price != 0 {
		db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "price"}, Value: c.Price})
	}
	if c.CreatedBy != 0 {
		db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "created_by"}, Value: c.CreatedBy})
	}
	if c.UpdatedBy != 0 {
		db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "updated_by"}, Value: c.UpdatedBy})
	}
	if c.DeletedBy != 0 {
		db.Where(clause.Eq{Column: clause.Column{Table: tableName, Name: "deleted_by"}, Value: c.DeletedBy})
	}
}

// ProductDao 是商品表的数据访问对象。
type ProductDao struct {
	*gormdao.Dao[model.ProductEntity, model.ProductEntityList, uint]
}

// NewProductDao 构造 ProductDao。
//
// 表名传**逻辑表名** model.TableNameProduct 即可：gormdao（v1.32.26 起）会把表名与列名
// 交给 GORM 按当前方言引用，调用方无需额外处理。
func NewProductDao() *ProductDao {
	return &ProductDao{
		Dao: gormdao.NewDao[model.ProductEntity, model.ProductEntityList, uint](
			model.TableNameProduct, "ProductDao",
			dbclient.DemoDB,
		),
	}
}
