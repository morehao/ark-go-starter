// Package code 集中注册业务错误码并统一返回 gerror.Error 值（见 GetError 注释）。
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
// 返回值刻意是 **gerror.Error 值**（不是 *gerror.Error）：
// gincontext.Fail 用 `var gErr gerror.Error; errors.As(err, &gErr)` 提取业务码，
// errors.As 要求错误链上的动态类型与目标类型可赋值——传指针时永远匹配不到，
// 结果 code 恒为 -1、msg 退化成 "[100305] xxx" 这种带内部码的文案。
// gerror.Error 的方法集全是值接收者，值本身即可满足 error 接口，传值不影响任何调用方。
//
// 传入未注册的错误码时 panic，让拼错错误码在开发/测试阶段立即暴露，
// 而不是静默返回 code:0 让接口看似成功。
func GetError(code int) gerror.Error {
	err, ok := errorMap[code]
	if !ok {
		panic(fmt.Sprintf("code.GetError: error code %d is not registered", code))
	}
	return err
}

func init() {
	// 业务错误码规范: 从 1001XX 开始
	// 模块划分: 1001XX(组织) 1002XX(租户) 1003XX(商品示例) 1004XX(部门) 1005XX(用户) 1006XX(菜单) 1007XX(角色) 1008XX(日志) 1009XX(API密钥)
	// 注: 100100-100109 被 application 使用
	registerError(commonErrorMsgMap)
	registerError(gconstant.DBErrorMsgMap)
	registerError(gconstant.SystemErrorMsgMap)
	registerError(gconstant.AuthErrorMsgMap)
	registerError(organizationErrorMsgMap)
	registerError(tenantErrorMsgMap)
	registerError(productErrorMsgMap)
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
