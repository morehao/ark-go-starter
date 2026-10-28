package dtouser

import (
	"github.com/morehao/ark-go-starter/demo/object/objuser"
	"github.com/morehao/ark-go-starter/pkg/operator"
	"github.com/morehao/golib/biz/gobject"
)

type UserCreateReq struct {
	objuser.UserBaseInfo
	operator.Operator
}

// UserCreateInTxReq 事务创建用户请求：在同一个事务里创建两条用户记录。
//
// FailAfterFirst 用于演示原子性：为 true 时，第一条插入成功后主动返回错误，
// 触发事务回滚，最终两条记录都不应落库。
type UserCreateInTxReq struct {
	FirstName      string `json:"firstName" binding:"required"`  // 第一条用户姓名（必填）
	SecondName     string `json:"secondName" binding:"required"` // 第二条用户姓名（必填）
	FailAfterFirst bool   `json:"failAfterFirst"`                // 是否在第一条后制造错误以演示回滚
}

// UserUpdateReq 修改用户管理请求。
//
// 业务字段用**指针**表达「是否传了该字段」：
//   - 指针为 nil 表示请求未携带该字段，service 侧不更新该列；
//   - 指针非 nil 表示显式赋值（含零值），service 侧按值更新。
//
// 这样既避免「客户端漏传字段把列清空」，也避免「传 0 被当作未传而无法清空」。
type UserUpdateReq struct {
	UserID       uint    `json:"-" uri:"userID" binding:"required"` // 自增 ID
	CompanyID    *uint   `json:"companyID" form:"companyID"`        // 公司id（nil 表示不修改）
	DepartmentID *uint   `json:"departmentID" form:"departmentID"`  // 部门id（nil 表示不修改）
	Name         *string `json:"name" form:"name"`                  // 姓名（nil 表示不修改）
	operator.Operator
}

type UserDetailReq struct {
	UserID uint `json:"-" uri:"userID" binding:"required"` // 自增 ID
}

// UserPageListReq 用户管理列表查询请求。
//
// 过滤字段与 dao.UserCond 一一对应，service 侧逐项映射；空值表示不按该条件过滤。
type UserPageListReq struct {
	gobject.PageQuery
	CompanyID    uint   `json:"companyID" form:"companyID"`       // 公司id（可选）
	DepartmentID uint   `json:"departmentID" form:"departmentID"` // 部门id（可选）
	Name         string `json:"name" form:"name"`                 // 姓名（可选，精确匹配）
}

type UserDeleteReq struct {
	UserID uint `json:"-" uri:"userID" binding:"required"` // 自增 ID
	operator.Operator
}
