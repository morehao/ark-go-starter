//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// settingResp 是配置项对外契约（对齐 golib ginsetting.settingResponse）。
type settingResp struct {
	ID             string `json:"id"`
	Group          string `json:"group"`
	Key            string `json:"key"`
	ValueType      string `json:"value_type"`
	Value          string `json:"value"`
	EncryptionMode string `json:"encryption_mode"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

// settingList 是配置项分页响应。
type settingList struct {
	List  []settingResp `json:"list"`
	Total int64         `json:"total"`
}

// findSetting 在列表中按 key 查找配置项。
func findSetting(t *testing.T, list settingList, key string) settingResp {
	t.Helper()
	for _, item := range list.List {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("setting %q not found in list", key)
	return settingResp{}
}

// TestSettingFlowOverHTTP 覆盖配置中心完整链路：
// 创建 → 分组/状态筛选 → 详情 → 局部更新（值/状态） → 同名 upsert → 删除（幂等）。
func TestSettingFlowOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	group := tag + "-setting-group"
	key := "e2e.setting.key"
	t.Cleanup(func() { dbExec(t, `DELETE FROM core_setting WHERE group_name = ?`, group) })

	// 1) 创建（json 值类型）。
	createResp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/settings", nil, map[string]any{
		"group":       group,
		"key":         key,
		"value_type":  "json",
		"value":       `{"a":1}`,
		"status":      "enabled",
		"description": "e2e setting",
	})
	require.Equal(t, 0, createResp.Code, createResp.Msg)

	// 2) 按分组筛选：能查到且只有一条。
	listResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/settings?group="+group+"&page=1&page_size=10", nil, nil)
	require.Equal(t, 0, listResp.Code, listResp.Msg)
	var byGroup settingList
	require.NoError(t, json.Unmarshal(listResp.Data, &byGroup))
	require.EqualValues(t, 1, byGroup.Total, "按唯一分组筛选应只命中 1 条")
	setting := findSetting(t, byGroup, key)
	assert.Equal(t, group, setting.Group)
	assert.Equal(t, "json", setting.ValueType)
	assert.Equal(t, "enabled", setting.Status)
	assert.Equal(t, "plain", setting.EncryptionMode)
	require.NotEmpty(t, setting.ID)

	// 3) 详情。
	detailResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/settings/"+setting.ID, nil, nil)
	require.Equal(t, 0, detailResp.Code, detailResp.Msg)
	var detail settingResp
	require.NoError(t, json.Unmarshal(detailResp.Data, &detail))
	assert.Equal(t, key, detail.Key)
	assert.JSONEq(t, `{"a":1}`, detail.Value)

	// 4) 更新值（局部更新：只传 value）。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPut, "/v1/demo/settings/"+setting.ID, nil, map[string]any{
		"value": `{"a":2}`,
	}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/settings/"+setting.ID, nil, nil).Data, &detail))
	assert.JSONEq(t, `{"a":2}`, detail.Value)

	// 5) 状态筛选：置为 disabled 后只应出现在 status=disabled 的结果里。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPut, "/v1/demo/settings/"+setting.ID, nil, map[string]any{
		"status": "disabled",
	}).Code)
	disabledList := doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/settings?group="+group+"&status=disabled&page=1&page_size=10", nil, nil)
	require.Equal(t, 0, disabledList.Code, disabledList.Msg)
	require.NoError(t, json.Unmarshal(disabledList.Data, &byGroup))
	require.EqualValues(t, 1, byGroup.Total, "停用后应能在 status=disabled 下查到")

	disabledList = doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/settings?group="+group+"&status=enabled&page=1&page_size=10", nil, nil)
	require.Equal(t, 0, disabledList.Code, disabledList.Msg)
	require.NoError(t, json.Unmarshal(disabledList.Data, &byGroup))
	assert.EqualValues(t, 0, byGroup.Total, "停用后不应再出现在 status=enabled 下")

	// 6) 同名 (group,key) 再次创建走 upsert：总数不变，值被覆盖。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPost, "/v1/demo/settings", nil, map[string]any{
		"group":      group,
		"key":        key,
		"value_type": "json",
		"value":      `{"a":3}`,
		"status":     "enabled",
	}).Code)
	upsertList := doJSON(t, baseURL, http.MethodGet, "/v1/demo/settings?group="+group+"&page=1&page_size=10", nil, nil)
	require.Equal(t, 0, upsertList.Code, upsertList.Msg)
	require.NoError(t, json.Unmarshal(upsertList.Data, &byGroup))
	require.EqualValues(t, 1, byGroup.Total, "同名 upsert 不应产生第二条记录")
	setting = findSetting(t, byGroup, key)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/settings/"+setting.ID, nil, nil).Data, &detail))
	assert.JSONEq(t, `{"a":3}`, detail.Value, "upsert 应用新值覆盖")
	assert.Equal(t, "enabled", detail.Status)

	// 7) 删除：先删一次成功，再删一次仍应成功（幂等）；之后详情报不存在。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/settings/"+setting.ID, nil, nil).Code)
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/settings/"+setting.ID, nil, nil).Code,
		"重复删除应幂等成功")
	assert.NotEqual(t, 0, doError(t, baseURL, http.MethodGet,
		"/v1/demo/settings/"+setting.ID, nil, nil).Code, "删除后详情应返回业务错误")
}

// TestSettingNegativeOverHTTP 覆盖配置中心的校验反例：非法值类型、值与类型不匹配、
// 缺必填字段、不存在的 ID。
func TestSettingNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	group := tag + "-setting-neg"
	t.Cleanup(func() { dbExec(t, `DELETE FROM core_setting WHERE group_name = ?`, group) })

	t.Run("unsupported_value_type_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/settings", nil, map[string]any{
			"group":      group,
			"key":        "e2e.neg.type",
			"value_type": "xml",
			"value":      "<a/>",
		})
		assert.NotEqual(t, 0, resp.Code, "不支持的值类型应返回业务错误")
	})

	t.Run("value_mismatched_with_type_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/settings", nil, map[string]any{
			"group":      group,
			"key":        "e2e.neg.value",
			"value_type": "int",
			"value":      "not-an-int",
		})
		assert.NotEqual(t, 0, resp.Code, "值与声明类型不匹配应返回业务错误")
	})

	t.Run("missing_value_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/settings", nil, map[string]any{
			"group": group,
			"key":   "e2e.neg.missing-value",
		})
		assert.NotEqual(t, 0, resp.Code, "缺 value 应在绑定阶段被拒")
	})

	t.Run("missing_group_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/settings", nil, map[string]any{
			"key":   "e2e.neg.missing-group",
			"value": "x",
		})
		assert.NotEqual(t, 0, resp.Code, "缺 group 应在绑定阶段被拒")
	})

	t.Run("detail_missing_id_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet,
			"/v1/demo/settings/00000000-0000-0000-0000-000000000000", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "不存在的配置项应返回业务错误")
	})
}
