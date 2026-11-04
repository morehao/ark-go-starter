package code

import "github.com/morehao/golib/gerror"

// API 密钥相关业务错误码。
const (
	APIKeyCreateError      = 101101
	APIKeyDeleteError      = 101102
	APIKeyUpdateError      = 101103
	APIKeyGetPageListError = 101104
	APIKeyNotExistError    = 101105
	APIKeyInvalidError     = 101106
	APIKeyExpiredError     = 101107
	APIKeyDisabledError    = 101108
)

//nolint:gosec // 错误码文案映射，非凭据。
var apiKeyErrorMsgMap = gerror.CodeMsgMap{
	APIKeyCreateError:      "创建API密钥失败",
	APIKeyDeleteError:      "删除API密钥失败",
	APIKeyUpdateError:      "更新API密钥失败",
	APIKeyGetPageListError: "获取API密钥列表失败",
	APIKeyNotExistError:    "API密钥不存在",
	APIKeyInvalidError:     "API密钥无效",
	APIKeyExpiredError:     "API密钥已过期",
	APIKeyDisabledError:    "API密钥已被禁用",
}
