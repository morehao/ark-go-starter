package code

import "github.com/morehao/golib/gerror"

// 商品管理相关业务错误码（1003XX 段）。
const (
	ProductCreateError      = 100300
	ProductDeleteError      = 100301
	ProductUpdateError      = 100302
	ProductGetDetailError   = 100303
	ProductGetPageListError = 100304
	ProductNotExistError    = 100305
)

var productErrorMsgMap = gerror.CodeMsgMap{
	ProductCreateError:      "创建商品管理失败",
	ProductDeleteError:      "删除商品管理失败",
	ProductUpdateError:      "修改商品管理失败",
	ProductGetDetailError:   "查看商品管理失败",
	ProductGetPageListError: "查看商品管理列表失败",
	ProductNotExistError:    "商品管理不存在",
}
