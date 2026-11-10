package code

import "github.com/morehao/golib/gerror"

// 1099XX 为仓库自留的通用 HTTP 层错误码段（1000XX~1011XX 已被 DB/组织/租户/商品/部门/用户/
// 菜单/角色/认证/日志等模块占用，勿复用）。
const (
	// RouteNotFoundError 请求的路由不存在（gin NoRoute），由引擎级统一 404 处理器返回。
	RouteNotFoundError = 109901
	// RouteMethodNotAllowedError 路径存在但 HTTP 方法不允许（gin NoMethod），
	// 由引擎级统一 405 处理器返回（Engine.HandleMethodNotAllowed = true 时才生效）。
	RouteMethodNotAllowedError = 109902
)

var commonErrorMsgMap = gerror.CodeMsgMap{
	RouteNotFoundError:         "接口不存在",
	RouteMethodNotAllowedError: "请求方法不允许",
}
