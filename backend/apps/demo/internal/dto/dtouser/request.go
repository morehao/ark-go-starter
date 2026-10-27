package dtouser

import (
	"github.com/morehao/ark-go-starter/demo/object/objuser"
	"github.com/morehao/golib/biz/gobject"
)

type UserCreateReq struct {
	objuser.UserBaseInfo
}

type UserUpdateReq struct {
	UserID uint `json:"-" uri:"userID" binding:"required"` // 自增 ID
	objuser.UserBaseInfo
}

type UserDetailReq struct {
	UserID uint `json:"-" uri:"userID" binding:"required"` // 自增 ID
}

type UserPageListReq struct {
	gobject.PageQuery
}

type UserDeleteReq struct {
	UserID     uint `json:"-" uri:"userID" binding:"required"` // 自增 ID
	OperatorID uint `json:"-"`                                 // 操作人 ID，由 controller 从登录态填充
}
