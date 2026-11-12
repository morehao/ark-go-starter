//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/morehao/ark-go-starter/apps/api/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// productDetail 是商品详情的断言视图（字段名对齐 dtoproduct.ProductDetailResp）。
type productDetail struct {
	ProductID  uint    `json:"productID"`
	Name       string  `json:"name"`
	CategoryID uint    `json:"categoryID"`
	Price      float64 `json:"price"`
	CreatedBy  uint64  `json:"createdBy"`
	UpdatedBy  uint64  `json:"updatedBy"`
	CreatedAt  int64   `json:"createdAt"`
	UpdatedAt  int64   `json:"updatedAt"`
}

// productPage 是商品分页响应的断言视图。
type productPage struct {
	List  []productDetail `json:"list"`
	Total int64           `json:"total"`
}

// createProduct 通过接口创建一个商品并注册物理清理，返回商品ID。
func createProduct(t *testing.T, baseURL, name string, categoryID uint, price float64, userID string) uint {
	t.Helper()
	resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/products",
		map[string]string{middleware.HeaderUserID: userID}, map[string]any{
			"name":       name,
			"categoryID": categoryID,
			"price":      price,
		})
	require.Equal(t, 0, resp.Code, "create product failed: %s", resp.Msg)
	var created struct {
		ProductID uint `json:"productID"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &created))
	require.NotZero(t, created.ProductID)
	t.Cleanup(func() { cleanupProductByName(t, name) })
	return created.ProductID
}

// TestProductPaginationFlow 覆盖真实端口下的分页与筛选：多条造数下 page/pageSize 生效、
// total 与实际命中数一致、名称精确筛选只返回目标商品。
func TestProductPaginationFlow(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	names := []string{tag + "-p1", tag + "-p2", tag + "-p3"}
	ids := make([]uint, 0, len(names))
	for i, name := range names {
		ids = append(ids, createProduct(t, baseURL, name, uint(40+i), 10.5+float64(i), "8810"))
	}

	// 第一页：pageSize=2 只返回 2 条，total 至少覆盖刚造的三条。
	page1 := doJSON(t, baseURL, http.MethodGet, "/v1/demo/products?page=1&pageSize=2", nil, nil)
	require.Equal(t, 0, page1.Code, page1.Msg)
	var first productPage
	require.NoError(t, json.Unmarshal(page1.Data, &first))
	assert.Len(t, first.List, 2, "pageSize=2 应只返回 2 条")
	assert.GreaterOrEqual(t, first.Total, int64(len(names)), "total 应不小于本次造数条数")

	// 第二页：仍受 pageSize 约束，且与第一页不重复（有稳定排序）。
	page2 := doJSON(t, baseURL, http.MethodGet, "/v1/demo/products?page=2&pageSize=2", nil, nil)
	require.Equal(t, 0, page2.Code, page2.Msg)
	var second productPage
	require.NoError(t, json.Unmarshal(page2.Data, &second))
	assert.LessOrEqual(t, len(second.List), 2, "pageSize=2 不应超过 2 条")
	if len(first.List) > 0 && len(second.List) > 0 {
		assert.NotEqual(t, first.List[0].ProductID, second.List[0].ProductID, "翻页不应重复首条")
	}

	// 名称精确筛选：只命中目标商品。
	filtered := doJSON(t, baseURL, http.MethodGet, "/v1/demo/products?page=1&pageSize=10&name="+names[1], nil, nil)
	require.Equal(t, 0, filtered.Code, filtered.Msg)
	var byName productPage
	require.NoError(t, json.Unmarshal(filtered.Data, &byName))
	require.Len(t, byName.List, 1, "按唯一名称筛选应只命中 1 条")
	assert.Equal(t, ids[1], byName.List[0].ProductID)
	assert.Equal(t, names[1], byName.List[0].Name)
}

// TestProductCreateInTxOverHTTP 覆盖事务示例接口的两条关键路径：
//   - failAfterFirst=false：两条记录都落库，返回两个 ID；
//   - failAfterFirst=true：第一条已写入后主动报错，整体回滚，库里两条都不存在。
func TestProductCreateInTxOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()

	t.Run("success_commits_both_rows", func(t *testing.T) {
		firstName := tag + "-tx-ok-1"
		secondName := tag + "-tx-ok-2"
		t.Cleanup(func() {
			cleanupProductByName(t, firstName)
			cleanupProductByName(t, secondName)
		})

		resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/products/tx", nil, map[string]any{
			"firstName":  firstName,
			"secondName": secondName,
		})
		require.Equal(t, 0, resp.Code, resp.Msg)

		var created struct {
			ProductIDs []uint `json:"productIDs"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &created))
		require.Len(t, created.ProductIDs, 2, "事务成功应返回两个商品ID")
		assert.EqualValues(t, 2, dbCount(t,
			`SELECT count(*) FROM "product" WHERE name IN (?, ?)`, firstName, secondName),
			"事务提交后两条记录都应落库")
	})

	t.Run("fail_after_first_rolls_back", func(t *testing.T) {
		firstName := tag + "-tx-rollback-1"
		secondName := tag + "-tx-rollback-2"
		t.Cleanup(func() {
			cleanupProductByName(t, firstName)
			cleanupProductByName(t, secondName)
		})

		resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/products/tx", nil, map[string]any{
			"firstName":      firstName,
			"secondName":     secondName,
			"failAfterFirst": true,
		})
		require.NotEqual(t, 0, resp.Code, "第一条后失败应返回业务错误，不能伪装成功")

		// 关键断言：第一条已 insert 成功，但事务回滚后两条都不应存在。
		assert.EqualValues(t, 0, dbCount(t,
			`SELECT count(*) FROM "product" WHERE name IN (?, ?)`, firstName, secondName),
			"事务回滚后两条记录都不应落库")
	})
}

// TestProductNegativeOverHTTP 覆盖商品模块的关键反例：不存在的 ID、非法路径参数、缺字段。
//
// 统一契约：业务错误走 HTTP 200 + 非 0 code（不得返回 500/空 body）。
func TestProductNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	t.Run("detail_missing_id_returns_business_error", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet, "/v1/demo/products/999999999", nil, nil)
		assert.EqualValues(t, 100305, resp.Code, "应命中 ProductNotExistError")
		assert.NotEmpty(t, resp.Msg)
	})

	t.Run("detail_non_numeric_id_is_bad_request", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet, "/v1/demo/products/not-a-number", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "非法路径参数应返回业务错误码")
	})

	t.Run("update_missing_id_returns_business_error", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPut, "/v1/demo/products/999999999", nil, map[string]any{
			"name": "e2e-should-not-exist",
		})
		assert.NotEqual(t, 0, resp.Code, "更新不存在的商品应返回业务错误码")
	})

	t.Run("delete_missing_id_returns_business_error", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodDelete, "/v1/demo/products/999999999", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "删除不存在的商品应返回业务错误码")
	})

	t.Run("tx_missing_required_field_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/products/tx", nil, map[string]any{
			"firstName": uniqueTag() + "-tx-missing",
		})
		assert.NotEqual(t, 0, resp.Code, "缺少 secondName 应在绑定阶段被拒")
	})

	t.Run("tx_empty_body_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/products/tx", nil, map[string]any{})
		assert.NotEqual(t, 0, resp.Code, "空 body 应在绑定阶段被拒")
	})
}
