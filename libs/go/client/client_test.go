package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestListProducts_GeneratedClient 演示 P5 试点生成的 Go client 可编译且可真实调用：
//   - 用 httptest 起一个假后端，返回与 contracts/openapi/openapi.json 约定的响应体；
//   - 通过生成的 ClientWithResponses.ListProductsWithResponse 发起请求并解析 200。
//
// 不依赖真实后端，属纯单元/可调用性验证（DoD：生成物可被依赖、可调用）。
func TestListProducts_GeneratedClient(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/demo/products" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"requestID":"test-req","msg":"success","data":{"list":[{"productID":1,"name":"demo","price":9.9,"categoryID":2,"createdAt":1700000000,"updatedAt":1700000000,"createdBy":9001,"updatedBy":9001}],"total":1}}`))
	}))
	defer srv.Close()

	c, err := NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatalf("NewClientWithResponses: %v", err)
	}

	page, pageSize, name := 1, 20, "demo"
	resp, err := c.ListProductsWithResponse(context.Background(), &ListProductsParams{
		Page:     &page,
		PageSize: &pageSize,
		Name:     &name,
	})
	if err != nil {
		t.Fatalf("ListProductsWithResponse: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode())
	}
	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		t.Fatalf("JSON200/data is nil: %s", string(resp.Body))
	}
	if resp.JSON200.Data.Total == nil || *resp.JSON200.Data.Total != 1 {
		t.Fatalf("want total=1, got %+v", resp.JSON200.Data.Total)
	}
	if gotQuery == "" {
		t.Fatalf("query params were not encoded")
	}
}
