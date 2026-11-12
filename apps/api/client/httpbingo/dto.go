package httpbingo

// GetRequest 是 Get 调用的请求参数。
type GetRequest struct {
	ID uint `json:"id" form:"id"`
}

// GetResponse 是 Get 调用的响应体。
type GetResponse struct {
	Args    GetArgs    `json:"args"`    // 入参
	Method  string     `json:"method"`  // 请求方法
	Origin  string     `json:"origin"`  // 请求来源
	URL     string     `json:"url"`     // 请求地址
	Headers GetHeaders `json:"headers"` // 请求头
}

// GetArgs 是 httpbingo /get 接口的查询参数。
type GetArgs struct {
	ID []string `json:"id"` // id
}

// GetHeaders 是 httpbingo /get 接口的请求头。
type GetHeaders struct {
	Accept         []string `json:"Accept"`
	AcceptEncoding []string `json:"Accept-Encoding"`
	AcceptLanguage []string `json:"Accept-Language"`
	Host           []string `json:"Host"`
}
