//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/morehao/ark-go-starter/demo/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoint 验证健康检查走真实端口时，Postgres 与 Redis 均为 ok。
func TestHealthEndpoint(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	resp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/health", nil, nil)
	require.Equal(t, 0, resp.Code, resp.Msg)

	var data struct {
		Postgres string `json:"postgres"`
		Redis    string `json:"redis"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "ok", data.Postgres, "postgres 健康检查应为 ok")
	assert.Equal(t, "ok", data.Redis, "redis 健康检查应为 ok")
}

// TestProductCRUDOverHTTP 覆盖真实 HTTP 端口下的商品增查改删全链路，并直连库核对审计列落库。
func TestProductCRUDOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	const creatorID uint64 = 8801
	const updaterID uint64 = 8802
	const deleterID uint64 = 8803

	// 创建：created_by/updated_by 应记创建人。
	createResp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/products",
		map[string]string{middleware.HeaderUserID: "8801"}, map[string]any{
			"name":       "e2e-product-crud",
			"categoryID": 22,
			"price":      19.90,
		})
	require.Equal(t, 0, createResp.Code, createResp.Msg)
	var created struct {
		ProductID uint `json:"productID"`
	}
	require.NoError(t, json.Unmarshal(createResp.Data, &created))
	require.NotZero(t, created.ProductID)
	t.Cleanup(func() { cleanupProduct(t, created.ProductID) })

	afterCreate := queryAudit(t, created.ProductID)
	assert.EqualValues(t, creatorID, afterCreate.CreatedBy)
	assert.EqualValues(t, creatorID, afterCreate.UpdatedBy)

	// 详情。
	detail := doJSON(t, baseURL, http.MethodGet, fmtProductPath(created.ProductID), nil, nil)
	assert.Equal(t, 0, detail.Code, detail.Msg)

	// 列表。
	list := doJSON(t, baseURL, http.MethodGet, "/v1/demo/products?page=1&pageSize=10", nil, nil)
	assert.Equal(t, 0, list.Code, list.Msg)

	// 修改：updated_by 换成修改人，created_by 不变。
	update := doJSON(t, baseURL, http.MethodPut, fmtProductPath(created.ProductID),
		map[string]string{middleware.HeaderUserID: "8802"}, map[string]any{
			"name":       "e2e-product-crud-updated",
			"categoryID": 33,
			"price":      29.90,
		})
	assert.Equal(t, 0, update.Code, update.Msg)
	afterUpdate := queryAudit(t, created.ProductID)
	assert.EqualValues(t, creatorID, afterUpdate.CreatedBy)
	assert.EqualValues(t, updaterID, afterUpdate.UpdatedBy)

	// 删除：deleted_by 等于删除人；软删除后详情查不到。
	del := doJSON(t, baseURL, http.MethodDelete, fmtProductPath(created.ProductID),
		map[string]string{middleware.HeaderUserID: "8803"}, nil)
	assert.Equal(t, 0, del.Code, del.Msg)
	afterDelete := queryAudit(t, created.ProductID)
	assert.EqualValues(t, deleterID, afterDelete.DeletedBy)

	gone := doJSON(t, baseURL, http.MethodGet, fmtProductPath(created.ProductID), nil, nil)
	assert.NotEqual(t, 0, gone.Code, "软删除后不应再查到")
}
