//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSwaggerDocs 覆盖 dev 环境下挂载在 /demo 前缀下的接口文档路由：
// swagger UI（/demo/docs/index.html）、原始 spec（/demo/docs/doc.json）与 reDoc（/demo/redocs）。
//
// 文档路由由 config.Conf.Server.Env == "dev" 决定是否注册（config.yaml 当前为 dev），
// 是「构建产物可直接交付联调」的一环，因此纳入 e2e。
func TestSwaggerDocs(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	t.Run("swagger_index", func(t *testing.T) {
		status, header, body := doRaw(t, baseURL, http.MethodGet, "/demo/docs/index.html", nil, nil)
		require.Equal(t, http.StatusOK, status, "swagger index 应返回 200")
		assert.Contains(t, strings.ToLower(string(header.Get("Content-Type"))), "text/html")
		assert.Contains(t, string(body), "swagger", "index.html 应包含 swagger UI")
	})

	t.Run("swagger_doc_json", func(t *testing.T) {
		status, _, body := doRaw(t, baseURL, http.MethodGet, "/demo/docs/doc.json", nil, nil)
		require.Equal(t, http.StatusOK, status, "doc.json 应返回 200")
		var spec struct {
			Swagger string         `json:"swagger"`
			Paths   map[string]any `json:"paths"`
		}
		require.NoError(t, json.Unmarshal(body, &spec), "doc.json 应是合法 JSON")
		assert.NotEmpty(t, spec.Swagger, "应包含 swagger 版本字段")
		assert.NotEmpty(t, spec.Paths, "接口文档应包含 paths")
		assert.Contains(t, spec.Paths, "/v1/demo/products", "商品接口应登记到文档")
	})

	t.Run("redocs", func(t *testing.T) {
		status, header, body := doRaw(t, baseURL, http.MethodGet, "/demo/redocs", nil, nil)
		require.Equal(t, http.StatusOK, status, "redocs 应返回 200")
		assert.Contains(t, strings.ToLower(string(header.Get("Content-Type"))), "text/html")
		assert.Contains(t, strings.ToLower(string(body)), "redoc", "redocs 页面应包含 reDoc")
	})
}
