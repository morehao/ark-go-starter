// Package svcexample 是示例服务层：演示响应格式化等能力。
package svcexample

import (
	"context"

	"github.com/morehao/ark-go-starter/apps/api/internal/dto/dtoexample"
)

// FormatSvc 是响应格式化示例服务接口。
type FormatSvc interface {
	FormatResponse(ctx context.Context) *dtoexample.FormatResponseResp
}

var _ FormatSvc = (*formatSvc)(nil)

type formatSvc struct{}

// NewFormatSvc 构造 FormatSvc。
func NewFormatSvc() FormatSvc {
	return &formatSvc{}
}

func (svc *formatSvc) FormatResponse(_ context.Context) *dtoexample.FormatResponseResp {
	pricePtr := 1.22245
	return &dtoexample.FormatResponseResp{
		Items: []dtoexample.FormatDataItem{
			{
				Price:     1.22245,
				PriceList: []float64{1.22245, 1.22255},
			},
		},
		FormatDataItem: dtoexample.FormatDataItem{
			Price:     1.22245,
			PriceList: []float64{1.22245, 1.22255},
		},
		ItemMap: map[string]dtoexample.FormatDataItem{
			"1": {
				Price:     1.22245,
				PriceList: []float64{1.22245, 1.22255},
			},
			"2": {
				Price: 1.22245,
			},
		},
		NameMap: map[string][]string{
			"a": {},
		},
		PriceList: []float64{1.22245, 1.22255},
		PriceMap: map[string]float64{
			"1": 1.22245,
		},
		PricePtr:   &pricePtr,
		AnyVal:     &dtoexample.FormatDataItem{Price: 1.22245, PriceList: []float64{1.22245}},
		AnyMap:     map[string]any{"k": &dtoexample.FormatDataItem{Price: 1.22245}},
		PtrItemMap: map[string]*dtoexample.FormatDataItem{"k": {Price: 1.22245}},
		NilSlice:   nil,
		NilMap:     nil,
		NoTagPrice: 1.22245,
	}
}
