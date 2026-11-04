package dtouser

import (
	"github.com/morehao/ark-go-starter/demo/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

// UserCreateResp 是创建用户的响应。
type UserCreateResp struct {
	UserID uint `json:"userID"` // 自增 ID
}

// UserCreateInTxResp 事务创建用户响应。
type UserCreateInTxResp struct {
	UserIDs []uint `json:"userIDs"` // 事务内创建成功的用户 ID 列表
}

// UserDetailResp 是用户详情的响应。
type UserDetailResp struct {
	UserID uint `json:"userID"` // 自增 ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

// UserPageListItem 是用户分页列表的单项。
type UserPageListItem struct {
	UserID uint `json:"userID"` // 自增 ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

// UserPageListResp 是用户分页列表的响应。
type UserPageListResp struct {
	List  []UserPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}
