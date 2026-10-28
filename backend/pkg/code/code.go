package code

import (
	"fmt"

	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gerror"
)

var errorMap = gerror.ErrorMap{}

func registerError(codeMsgMap gerror.CodeMsgMap) {
	for code, msg := range codeMsgMap {

		if _, ok := errorMap[code]; ok {
			panic(fmt.Sprintf("error code %d already exists", code))
		}
		errorMap[code] = gerror.Error{
			Code: code,
			Msg:  msg,
		}
	}
}

// GetError 返回指定错误码对应的业务错误。
//
// 传入未注册的错误码时，不返回零值 Error{0, ""}——那会被 gincontext.Fail 当作
// 「code:0 成功码」静默响应，导致拼错错误码时接口看似成功。这里兜底为系统错误码，
// 并 panic，让问题在开发/测试阶段立即暴露，而不是上线后静默返回成功。
func GetError(code int) *gerror.Error {
	err, ok := errorMap[code]
	if !ok {
		panic(fmt.Sprintf("code.GetError: error code %d is not registered", code))
	}
	return &err
}

func init() {
	// 业务错误码规范: 从 1001XX 开始
	// 模块划分: 1001XX(组织) 1002XX(租户) 1004XX(部门) 1005XX(用户) 1006XX(菜单) 1007XX(角色) 1008XX(日志) 1009XX(API密钥)
	// 注: 100100-100109 被 application 使用
	registerError(gconstant.DBErrorMsgMap)
	registerError(gconstant.SystemErrorMsgMap)
	registerError(gconstant.AuthErrorMsgMap)
	registerError(organizationErrorMsgMap)
	registerError(tenantErrorMsgMap)
	registerError(departmentErrorMsgMap)
	registerError(menuErrorMsgMap)
	registerError(roleErrorMsgMap)
	registerError(authErrorMsgMap)
	registerError(applicationErrorMsgMap)
	registerError(operationLogErrorMsgMap)
	registerError(loginLogErrorMsgMap)
	registerError(apiKeyErrorMsgMap)
	registerError(oidcErrorMsgMap)
	registerError(userErrorMsgMap)
}
