package dtoexample

import "fmt"

// FormatResponseResp 数值精度格式化演示响应体，覆盖各类型字段的格式化输出
type FormatResponseResp struct {
	Items          []FormatDataItem           `json:"items"` // 结构体切片
	FormatDataItem                            // 内嵌结构体，字段平铺到响应体
	ItemMap        map[string]FormatDataItem  `json:"itemMap"`                 // 结构体 map
	PriceList      []float64                  `json:"priceList" precision:"2"` // 小数切片（保留 2 位小数）
	NameList       []string                   `json:"nameList"`                // 字符串切片
	NameMap        map[string][]string        `json:"nameMap"`                 // 字符串切片 map
	PriceMap       map[string]float64         `json:"priceMap" precision:"2"`  // 小数 map（保留 2 位小数）
	PricePtr       *float64                   `json:"pricePtr" precision:"2"`  // 小数指针（保留 2 位小数）
	AnyVal         any                        `json:"anyVal"`                  // 任意类型字段
	AnyMap         map[string]any             `json:"anyMap"`                  // 任意类型 map
	PtrItemMap     map[string]*FormatDataItem `json:"ptrItemMap"`              // 结构体指针 map
	NilSlice       []float64                  `json:"nilSlice" precision:"2"`  // 空切片（nil，输出保持 null）
	NilMap         map[string]float64         `json:"nilMap" precision:"2"`    // 空 map（nil，输出保持 null）
	NoTagPrice     float64                    `json:"noTagPrice"`              // 未标注 precision 的小数（原值输出）
}

// FormatDataItem 精度格式化演示中的嵌套数据结构
type FormatDataItem struct {
	Price     float64          `json:"price" precision:"2"`     // 小数（保留 2 位小数）
	PriceList []float64        `json:"priceList" precision:"2"` // 小数切片（保留 2 位小数）
	DescList  []string         `json:"descList"`                // 描述列表
	Children  []FormatDataItem `json:"children"`                // 子节点（递归结构）
}

// SSEMessage SSE 推送消息
type SSEMessage struct {
	ID    string `json:"id"`    // 消息 ID
	Event string `json:"event"` // 事件名
	Data  string `json:"data"`  // 数据体
}

func (msg SSEMessage) Format() string {
	result := ""
	if msg.ID != "" {
		result += fmt.Sprintf("id: %s\n", msg.ID)
	}
	if msg.Event != "" {
		result += fmt.Sprintf("event: %s\n", msg.Event)
	}
	result += fmt.Sprintf("data: %s\n\n", msg.Data)
	return result
}
