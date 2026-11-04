package code

import "github.com/morehao/golib/gerror"

// 操作日志与登录日志相关业务错误码。
const (
	OperationLogCreateError      = 101001
	OperationLogGetPageListError = 101002
	LoginLogCreateError          = 101003
	LoginLogGetPageListError     = 101004
)

var operationLogErrorMsgMap = gerror.CodeMsgMap{
	OperationLogCreateError:      "创建操作日志失败",
	OperationLogGetPageListError: "获取操作日志列表失败",
}

var loginLogErrorMsgMap = gerror.CodeMsgMap{
	LoginLogCreateError:      "创建登录日志失败",
	LoginLogGetPageListError: "获取登录日志列表失败",
}
