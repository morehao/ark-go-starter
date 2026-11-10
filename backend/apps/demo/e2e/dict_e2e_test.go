//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dictTypeResp 是字典类型对外契约（对齐 golib gindict.dictTypeResponse）。
type dictTypeResp struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Extra       string `json:"extra"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// dictItemResp 是字典项对外契约（对齐 golib gindict.dictItemResponse）。
type dictItemResp struct {
	ID          string `json:"id"`
	TypeCode    string `json:"type_code"`
	Value       string `json:"value"`
	Label       string `json:"label"`
	ParentID    string `json:"parent_id"`
	Level       int    `json:"level"`
	Sort        int    `json:"sort"`
	Status      string `json:"status"`
	Extra       string `json:"extra"`
	Description string `json:"description"`
}

// dictTreeNode 是树节点：项字段平铺 + children。
type dictTreeNode struct {
	dictItemResp
	Children []dictTreeNode `json:"children"`
}

// dictItemList 是项列表响应。
type dictItemList struct {
	List []dictItemResp `json:"list"`
}

// dictTypeList 是类型分页响应。
type dictTypeList struct {
	List  []dictTypeResp `json:"list"`
	Total int64          `json:"total"`
}

// dictIntegrity 是巡检响应。
type dictIntegrity struct {
	TypeCode string   `json:"type_code"`
	Checked  int      `json:"checked"`
	Healthy  bool     `json:"healthy"`
	Issues   []any    `json:"issues"`
	Orphans  []string `json:"orphans"`
}

// createDictType 通过接口创建字典类型并注册清理（按 code 硬删，含项）。
//
// 注意：dict 的 code 有字符集校验（[A-Za-z0-9_.:-]），不允许 '-'，
// 因此码值用 e2e_dict_* 下划线形式，展示名仍保留标准 e2e- 前缀。
func createDictType(t *testing.T, baseURL, code, name string) string {
	t.Helper()
	resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/dict-types", nil, map[string]any{
		"code":        code,
		"name":        name,
		"description": "e2e dict type",
	})
	require.Equal(t, 0, resp.Code, "create dict type failed: %s", resp.Msg)
	var created dictTypeResp
	require.NoError(t, json.Unmarshal(resp.Data, &created))
	require.NotEmpty(t, created.ID)

	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_dict_item WHERE type_code = ?`, code)
		dbExec(t, `DELETE FROM core_dict_type WHERE code = ?`, code)
	})
	return created.ID
}

// createDictItem 通过接口创建字典项，返回项ID。
func createDictItem(t *testing.T, baseURL, typeCode, value, label, parentID string) string {
	t.Helper()
	resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/dict-items", nil, map[string]any{
		"type_code": typeCode,
		"value":     value,
		"label":     label,
		"parent_id": parentID,
	})
	require.Equal(t, 0, resp.Code, "create dict item failed: %s", resp.Msg)
	var created dictItemResp
	require.NoError(t, json.Unmarshal(resp.Data, &created))
	require.NotEmpty(t, created.ID)
	return created.ID
}

// getDictItem 读取单个项（经 items 列表按 value 反查，接口未提供按 ID 的详情端点）。
func getDictItem(t *testing.T, baseURL, typeCode, value string) dictItemResp {
	t.Helper()
	resp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-codes/"+typeCode+"/items?include_disabled=1", nil, nil)
	require.Equal(t, 0, resp.Code, resp.Msg)
	var list dictItemList
	require.NoError(t, json.Unmarshal(resp.Data, &list))
	for _, item := range list.List {
		if item.Value == value {
			return item
		}
	}
	t.Fatalf("dict item %q not found under type %q", value, typeCode)
	return dictItemResp{}
}

// TestDictFlowOverHTTP 覆盖数据字典管理面的完整链路：
// 类型 CRUD/启停 → 项 CRUD/启停 → parent 移动 → items/children/tree → 巡检 →
// 「有子项拒绝删除」→ 级联删除与最终删除。
func TestDictFlowOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	code := uniqueDictToken("e2e_dict")
	parentValue := uniqueDictToken("e2e_parent")
	childValue := uniqueDictToken("e2e_child")

	// 1) 类型：创建 → 列表筛选 → 详情。
	typeID := createDictType(t, baseURL, code, tag+"-dict-type")

	listResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-types?code="+code, nil, nil)
	require.Equal(t, 0, listResp.Code, listResp.Msg)
	var types dictTypeList
	require.NoError(t, json.Unmarshal(listResp.Data, &types))
	require.EqualValues(t, 1, types.Total, "按唯一 code 筛选应只命中 1 条")
	require.Len(t, types.List, 1)
	assert.Equal(t, code, types.List[0].Code)
	assert.Equal(t, "enabled", types.List[0].Status)

	detailResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-types/"+typeID, nil, nil)
	require.Equal(t, 0, detailResp.Code, detailResp.Msg)
	var typeDetail dictTypeResp
	require.NoError(t, json.Unmarshal(detailResp.Data, &typeDetail))
	assert.Equal(t, code, typeDetail.Code)

	// 2) 类型更新（指针字段语义：只改 name/description）。
	updatedName := tag + "-dict-type-updated"
	updateResp := doJSON(t, baseURL, http.MethodPut, "/v1/demo/dict-types/"+typeID, nil, map[string]any{
		"name":        updatedName,
		"description": "e2e dict type updated",
	})
	require.Equal(t, 0, updateResp.Code, updateResp.Msg)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-types/"+typeID, nil, nil).Data, &typeDetail))
	assert.Equal(t, updatedName, typeDetail.Name)

	// 3) 类型启停。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/dict-types/"+typeID+"/status", nil, map[string]any{"status": "disabled"}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-types/"+typeID, nil, nil).Data, &typeDetail))
	assert.Equal(t, "disabled", typeDetail.Status, "停用后详情应反映 disabled")
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/dict-types/"+typeID+"/status", nil, map[string]any{"status": "enabled"}).Code)
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-types/"+typeID, nil, nil).Data, &typeDetail))
	assert.Equal(t, "enabled", typeDetail.Status)

	// 4) 项：父项 + 子项。
	parentID := createDictItem(t, baseURL, code, parentValue, tag+"-父项", "")
	childID := createDictItem(t, baseURL, code, childValue, tag+"-子项", parentID)

	childItem := getDictItem(t, baseURL, code, childValue)
	assert.Equal(t, parentID, childItem.ParentID)
	assert.Equal(t, 2, childItem.Level, "子项层级应为 2")

	// 5) items：两条都在；children：根节点只有父项，父节点下只有子项。
	itemsResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-codes/"+code+"/items", nil, nil)
	require.Equal(t, 0, itemsResp.Code, itemsResp.Msg)
	var items dictItemList
	require.NoError(t, json.Unmarshal(itemsResp.Data, &items))
	assert.Len(t, items.List, 2, "类型下应有 2 个项")

	rootChildren := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-codes/"+code+"/children", nil, nil)
	require.Equal(t, 0, rootChildren.Code, rootChildren.Msg)
	var rootList dictItemList
	require.NoError(t, json.Unmarshal(rootChildren.Data, &rootList))
	require.Len(t, rootList.List, 1, "根层应只有父项")
	assert.Equal(t, parentID, rootList.List[0].ID)

	subChildren := doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-codes/"+code+"/children?parent_id="+parentID, nil, nil)
	require.Equal(t, 0, subChildren.Code, subChildren.Msg)
	var subList dictItemList
	require.NoError(t, json.Unmarshal(subChildren.Data, &subList))
	require.Len(t, subList.List, 1, "父节点下应只有子项")
	assert.Equal(t, childID, subList.List[0].ID)

	// 6) tree：一个根节点，带一个子节点。
	treeResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-codes/"+code+"/tree", nil, nil)
	require.Equal(t, 0, treeResp.Code, treeResp.Msg)
	var tree []dictTreeNode
	require.NoError(t, json.Unmarshal(treeResp.Data, &tree))
	require.Len(t, tree, 1, "树应只有一个根节点")
	assert.Equal(t, parentID, tree[0].ID)
	require.Len(t, tree[0].Children, 1, "根节点下应有 1 个子节点")
	assert.Equal(t, childID, tree[0].Children[0].ID)

	// 7) 项更新（label/sort）。
	updateItemResp := doJSON(t, baseURL, http.MethodPut, "/v1/demo/dict-items/"+childID, nil, map[string]any{
		"label": tag + "-子项改",
		"sort":  9,
	})
	require.Equal(t, 0, updateItemResp.Code, updateItemResp.Msg)
	childItem = getDictItem(t, baseURL, code, childValue)
	assert.Equal(t, tag+"-子项改", childItem.Label)
	assert.Equal(t, 9, childItem.Sort)

	// 8) parent 移动：子项提升为根 → 根层 2 条；再挂回父项。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPut,
		"/v1/demo/dict-items/"+childID+"/parent", nil, map[string]any{"parent_id": ""}).Code)
	rootList = dictItemList{}
	require.NoError(t, json.Unmarshal(doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-codes/"+code+"/children", nil, nil).Data, &rootList))
	assert.Len(t, rootList.List, 2, "子项提升为根后根层应有 2 条")

	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPut,
		"/v1/demo/dict-items/"+childID+"/parent", nil, map[string]any{"parent_id": parentID}).Code)
	childItem = getDictItem(t, baseURL, code, childValue)
	assert.Equal(t, parentID, childItem.ParentID, "挂回后 parent 应为父项")
	assert.Equal(t, 2, childItem.Level)

	// 9) 项启停：默认列表不含停用项，include_disabled=1 才含。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/dict-items/"+childID+"/status", nil, map[string]any{"status": "disabled"}).Code)
	enabledOnly := doJSON(t, baseURL, http.MethodGet, "/v1/demo/dict-codes/"+code+"/items", nil, nil)
	require.Equal(t, 0, enabledOnly.Code, enabledOnly.Msg)
	require.NoError(t, json.Unmarshal(enabledOnly.Data, &items))
	assert.Len(t, items.List, 1, "默认不返回停用项")
	withDisabled := doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-codes/"+code+"/items?include_disabled=1", nil, nil)
	require.Equal(t, 0, withDisabled.Code, withDisabled.Msg)
	require.NoError(t, json.Unmarshal(withDisabled.Data, &items))
	assert.Len(t, items.List, 2, "include_disabled=1 应返回停用项")
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodPatch,
		"/v1/demo/dict-items/"+childID+"/status", nil, map[string]any{"status": "enabled"}).Code)

	// 10) 巡检：结构健康、无孤儿。
	integrityResp := doJSON(t, baseURL, http.MethodGet,
		"/v1/demo/dict-types/"+typeID+"/integrity-check", nil, nil)
	require.Equal(t, 0, integrityResp.Code, integrityResp.Msg)
	var integrity dictIntegrity
	require.NoError(t, json.Unmarshal(integrityResp.Data, &integrity))
	assert.Equal(t, code, integrity.TypeCode)
	assert.True(t, integrity.Healthy, "正常结构应巡检健康")
	assert.GreaterOrEqual(t, integrity.Checked, 2)
	assert.Empty(t, integrity.Orphans)

	// 11) 删除守卫：有子项的父项、有项的类型，默认都拒绝删除。
	assert.NotEqual(t, 0, doError(t, baseURL, http.MethodDelete,
		"/v1/demo/dict-items/"+parentID, nil, nil).Code, "父项仍有子项，删除应被拒")
	assert.NotEqual(t, 0, doError(t, baseURL, http.MethodDelete,
		"/v1/demo/dict-types/"+typeID, nil, nil).Code, "类型下仍有项，删除应被拒（未级联）")

	// 12) 自底向上删除：先子后父，再删类型。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/dict-items/"+childID, nil, nil).Code)
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/dict-items/"+parentID, nil, nil).Code)
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/dict-types/"+typeID, nil, nil).Code,
		"项已清空后删除类型应成功")

	assert.NotEqual(t, 0, doError(t, baseURL, http.MethodGet,
		"/v1/demo/dict-types/"+typeID, nil, nil).Code, "类型删除后详情应返回不存在")
	assert.EqualValues(t, 0, dbCount(t, `SELECT count(*) FROM core_dict_type WHERE code = ?`, code))
}

// TestDictCascadeDelete 覆盖级联删除：?cascade=true 时类型与其下所有项一起删除。
func TestDictCascadeDelete(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	code := uniqueDictToken("e2e_dict_cascade")
	typeID := createDictType(t, baseURL, code, tag+"-dict-cascade")
	createDictItem(t, baseURL, code, uniqueDictToken("e2e_cascade_item"), tag+"-项", "")

	resp := doJSON(t, baseURL, http.MethodDelete, "/v1/demo/dict-types/"+typeID+"?cascade=true", nil, nil)
	require.Equal(t, 0, resp.Code, resp.Msg)
	assert.EqualValues(t, 0, dbCount(t, `SELECT count(*) FROM core_dict_item WHERE type_code = ?`, code),
		"级联删除后项应全部清空")
	assert.EqualValues(t, 0, dbCount(t, `SELECT count(*) FROM core_dict_type WHERE code = ?`, code))
}

// TestDictNegativeOverHTTP 覆盖字典模块的校验反例：非法码值、重复码值、非法状态、
// 不存在的类型/项，统一契约是 HTTP 200 + 非 0 code。
func TestDictNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	code := uniqueDictToken("e2e_dict_neg")
	typeID := createDictType(t, baseURL, code, tag+"-dict-neg")

	t.Run("create_type_missing_code_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/dict-types", nil, map[string]any{
			"name": tag + "-no-code",
		})
		assert.NotEqual(t, 0, resp.Code, "缺 code 应在绑定阶段被拒")
	})

	t.Run("create_type_invalid_code_charset_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/dict-types", nil, map[string]any{
			"code": "e2e invalid code!",
			"name": tag + "-bad-code",
		})
		assert.NotEqual(t, 0, resp.Code, "非法 code 字符集应返回业务错误")
	})

	t.Run("create_type_duplicate_code_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/dict-types", nil, map[string]any{
			"code": code,
			"name": tag + "-dup-code",
		})
		assert.NotEqual(t, 0, resp.Code, "重复 code 应返回业务错误")
	})

	t.Run("create_item_under_missing_type_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/dict-items", nil, map[string]any{
			"type_code": uniqueDictToken("e2e_missing_type"),
			"value":     uniqueDictToken("e2e_item"),
			"label":     tag + "-项",
		})
		assert.NotEqual(t, 0, resp.Code, "类型不存在时建项应返回业务错误")
	})

	t.Run("create_item_invalid_value_charset_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/dict-items", nil, map[string]any{
			"type_code": code,
			"value":     "e2e bad value!",
			"label":     tag + "-项",
		})
		assert.NotEqual(t, 0, resp.Code, "非法 value 字符集应返回业务错误")
	})

	t.Run("create_item_missing_label_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/dict-items", nil, map[string]any{
			"type_code": code,
			"value":     uniqueDictToken("e2e_item"),
		})
		assert.NotEqual(t, 0, resp.Code, "缺 label 应在绑定阶段被拒")
	})

	t.Run("invalid_type_status_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPatch,
			"/v1/demo/dict-types/"+typeID+"/status", nil, map[string]any{"status": "archived"})
		assert.NotEqual(t, 0, resp.Code, "非法状态值应返回业务错误")
	})

	t.Run("detail_missing_type_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet,
			"/v1/demo/dict-types/00000000-0000-0000-0000-000000000000", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "不存在的类型应返回业务错误")
	})

	t.Run("move_item_into_descendant_rejected", func(t *testing.T) {
		parentValue := uniqueDictToken("e2e_cycle_parent")
		childValue := uniqueDictToken("e2e_cycle_child")
		parentID := createDictItem(t, baseURL, code, parentValue, tag+"-环父", "")
		childID := createDictItem(t, baseURL, code, childValue, tag+"-环子", parentID)

		// 把父项挂到自己的子项下应被拒（成环）。
		resp := doError(t, baseURL, http.MethodPut,
			"/v1/demo/dict-items/"+parentID+"/parent", nil, map[string]any{"parent_id": childID})
		assert.NotEqual(t, 0, resp.Code, "移入自身子树应返回业务错误")
	})
}
