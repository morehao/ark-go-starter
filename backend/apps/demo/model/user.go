package model

import (
	"gorm.io/gorm"
)

// UserEntity 用户管理表结构体
//
// tag 里只写 Go 类型 + size/default/comment，让 gorm 的 PostgreSQL 方言决定列类型
// （uint → bigint、string + size:255 → varchar(255)）。禁止写其它方言的字面量类型：
// `type:bigint unsigned` 会被原样下发给 PG，建表直接报 syntax error at or near "unsigned"。
type UserEntity struct {
	gorm.Model
	CompanyID    uint   `gorm:"column:company_id;not null;default:0;comment:公司id"`
	DepartmentID uint   `gorm:"column:department_id;not null;default:0;comment:部门id"`
	Name         string `gorm:"column:name;size:255;not null;default:'';comment:姓名"`
	CreatedBy    uint   `gorm:"column:created_by;not null;default:0;comment:创建人id"`
	UpdatedBy    uint   `gorm:"column:updated_by;not null;default:0;comment:更新人id"`
	DeletedBy    uint   `gorm:"column:deleted_by;not null;default:0;comment:删除人id"`
}

type UserEntityList []UserEntity

func (UserEntity) TableName() string {
	return TableNameUser
}

func (l UserEntityList) ToMap() map[uint]UserEntity {
	m := make(map[uint]UserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
