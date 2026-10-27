package objuser

// Operator 当前请求的操作者身份。
//
// 由 controller 从登录态取出后填入请求 DTO，service / dao 只读 DTO，不接触 gin 上下文。
// 字段带 json:"-"，客户端无法通过请求体伪造操作人。
type Operator struct {
	UserID uint `json:"-"` // 操作人用户ID
}
