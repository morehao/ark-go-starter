package model

import (
	"gorm.io/gorm"
)

// ProductEntity 商品管理表结构体
//
// 每个字段显式写 PG 的 type，与 apps/api/db/table_init.sql 一一对应（结构体自描述，
// 读模型即知列类型）。type 必须是 PostgreSQL 合法类型：gorm 只把 bool/int/uint/float/string/
// time/bytes 这类逻辑类型名交给方言映射，其它取值原样拼进 DDL，写成其它方言的字面量
// （如 type:bigint unsigned）在 PG 建表会直接 syntax error at or near "unsigned"。
type ProductEntity struct {
	gorm.Model
	Name       string  `gorm:"column:name;type:varchar(255);not null;default:'';comment:商品名称"`
	CategoryID uint    `gorm:"column:category_id;type:bigint;not null;default:0;comment:分类id"`
	Price      float64 `gorm:"column:price;type:numeric;precision:10;scale:2;not null;default:0;comment:价格"`
	CreatedBy  uint    `gorm:"column:created_by;type:bigint;not null;default:0;comment:创建人id"`
	UpdatedBy  uint    `gorm:"column:updated_by;type:bigint;not null;default:0;comment:更新人id"`
	DeletedBy  uint    `gorm:"column:deleted_by;type:bigint;not null;default:0;comment:删除人id"`
}

// ProductEntityList 是 ProductEntity 的切片类型。
type ProductEntityList []ProductEntity

// TableName 返回商品表的物理表名。
func (ProductEntity) TableName() string {
	return TableNameProduct
}

// ToMap 把列表转成以主键为 key 的 map。
func (l ProductEntityList) ToMap() map[uint]ProductEntity {
	m := make(map[uint]ProductEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
