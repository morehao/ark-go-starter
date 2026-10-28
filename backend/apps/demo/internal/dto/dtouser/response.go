package dtouser

import (
	"github.com/morehao/ark-go-starter/demo/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserCreateResp struct {
	UserID uint `json:"userID"` // 自增 ID
}

// UserCreateInTxResp 事务创建用户响应。
type UserCreateInTxResp struct {
	UserIDs []uint `json:"userIDs"` // 事务内创建成功的用户 ID 列表
}

type UserDetailResp struct {
	UserID uint `json:"userID"` // 自增 ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

type UserPageListItem struct {
	UserID uint `json:"userID"` // 自增 ID
	objuser.UserBaseInfo
	gobject.OperatorBaseInfo
}

type UserPageListResp struct {
	List  []UserPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}
