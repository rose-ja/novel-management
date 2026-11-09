//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

type categoryPayload struct {
	ID             string `json:"id"`
	WorkID         string `json:"work_id"`
	Name           string `json:"name"`
	CategoryType   string `json:"category_type"`
	IsBuiltin      bool   `json:"is_builtin"`
	IsEnabled      bool   `json:"is_enabled"`
	SortOrder      int    `json:"sort_order"`
	NodeCount      int64  `json:"node_count"`
	ReferenceCount int64  `json:"reference_count"`
	Version        int    `json:"version"`
}

type categoryListPayload struct {
	Items []categoryPayload `json:"items"`
}

type categoryDeletePayload struct {
	Deleted       bool   `json:"deleted"`
	UndoToken     string `json:"undo_token"`
	AffectedNodes int64  `json:"affected_nodes"`
}

func (h *harness) listCategories(workID string, includeDisabled *bool) categoryListPayload {
	h.t.Helper()
	path := "/api/v1/system-categories?work_id=" + workID
	if includeDisabled != nil {
		if *includeDisabled {
			path += "&include_disabled=true"
		} else {
			path += "&include_disabled=false"
		}
	}
	rec := h.do(http.MethodGet, path, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取体系类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[categoryListPayload](h.t, rec)
}

func (h *harness) createCategory(workID, name, categoryType string) categoryPayload {
	h.t.Helper()
	rec := h.do(http.MethodPost, "/api/v1/system-categories", map[string]any{
		"work_id":       workID,
		"name":          name,
		"category_type": categoryType,
	}, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[categoryPayload](h.t, rec)
}

func systemRepoForTest(h *harness) *repository.System {
	return repository.NewSystem(h.tx)
}

func TestSystemCategoriesListAndCreate(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("体系类别")

	list := h.listCategories(work.ID, nil)
	if len(list.Items) != 1 || !list.Items[0].IsBuiltin || list.Items[0].Name != model.BuiltinRealmCategory {
		t.Fatalf("初始应只有内置境界：%+v", list.Items)
	}
	if list.Items[0].SortOrder != 0 || list.Items[0].Version != 1 {
		t.Fatalf("内置类别字段不符：%+v", list.Items[0])
	}

	rec := h.do(http.MethodGet, "/api/v1/system-categories", nil, nil)
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	var details struct {
		Field string `json:"field"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil || details.Field != "work_id" {
		t.Fatalf("缺少 work_id 应指向字段 work_id：%s", envelope.Error.Details)
	}

	era := h.createCategory(work.ID, "纪元", model.CategoryTypeEra)
	if era.CategoryType != model.CategoryTypeEra || era.IsBuiltin || !era.IsEnabled {
		t.Fatalf("新建类别字段不符：%+v", era)
	}
	if era.SortOrder != 1 || era.Version != 1 {
		t.Fatalf("新建类别排序/版本不符：%+v", era)
	}

	h.createCategory(work.ID, "ERA", model.CategoryTypeNormal)
	rec = h.do(http.MethodPost, "/api/v1/system-categories", map[string]any{
		"work_id":       work.ID,
		"name":          "ｅｒａ",
		"category_type": model.CategoryTypeNormal,
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "NAME_CONFLICT")

	rec = h.do(http.MethodPost, "/api/v1/system-categories", map[string]any{
		"work_id":       work.ID,
		"name":          "错误类型",
		"category_type": "realm",
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	list = h.listCategories(work.ID, nil)
	if len(list.Items) != 3 {
		t.Fatalf("应有内置+纪元+ERA 共 3 条，实际 %d", len(list.Items))
	}
	if !list.Items[0].IsBuiltin {
		t.Fatalf("内置类别应置顶：%+v", list.Items[0])
	}
}

func TestSystemCategoriesBuiltinProtection(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("内置保护")
	builtin := h.listCategories(work.ID, nil).Items[0]

	rec := h.do(http.MethodPatch, "/api/v1/system-categories/"+builtin.ID, map[string]any{
		"name": "境界改名",
	}, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_BUILTIN_IMMUTABLE")

	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+builtin.ID, map[string]any{
		"is_enabled": false,
	}, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_BUILTIN_IMMUTABLE")

	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+builtin.ID, map[string]any{
		"is_enabled": true,
	}, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("内置 is_enabled=true 应视为无操作（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if got := decodeBody[categoryPayload](t, rec); got.Version != 1 {
		t.Fatalf("无操作不应改变版本：%+v", got)
	}

	rec = h.do(http.MethodDelete, "/api/v1/system-categories/"+builtin.ID, nil, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_BUILTIN_IMMUTABLE")

	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+builtin.ID, map[string]any{
		"category_type": "era",
	}, map[string]string{"If-Match": `"1"`})
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	var details struct {
		Field string `json:"field"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil || details.Field != "category_type" {
		t.Fatalf("category_type 不可改应指向字段 category_type：%s", envelope.Error.Details)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+builtin.ID, map[string]any{
		"name": "境界改名",
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestSystemCategoriesRenameAndDisable(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("类别启停")
	category := h.createCategory(work.ID, "血统", model.CategoryTypeNormal)

	rec := h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{
		"is_enabled": false,
	}, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("停用类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	disabled := decodeBody[categoryPayload](t, rec)
	if disabled.IsEnabled || disabled.Version != 2 {
		t.Fatalf("停用结果不符：%+v", disabled)
	}

	all := h.listCategories(work.ID, nil)
	if len(all.Items) != 2 {
		t.Fatalf("默认 include_disabled=true 应返回 2 条，实际 %d", len(all.Items))
	}
	enabledOnly := false
	only := h.listCategories(work.ID, &enabledOnly)
	if len(only.Items) != 1 || only.Items[0].ID != all.Items[0].ID {
		t.Fatalf("include_disabled=false 应只留内置：%+v", only.Items)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{
		"name":       "血统改名",
		"is_enabled": false,
	}, map[string]string{"If-Match": `"2"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("改名失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	renamed := decodeBody[categoryPayload](t, rec)
	if renamed.Name != "血统改名" || renamed.IsEnabled || renamed.Version != 3 {
		t.Fatalf("改名结果不符：%+v", renamed)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{
		"name": "其他",
	}, map[string]string{"If-Match": `"2"`})
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestSystemCategoriesLimit(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("类别上限")
	for i := 0; i < model.MaxSystemCategories-1; i++ {
		h.createCategory(work.ID, "类别"+string(rune('A'+i)), model.CategoryTypeNormal)
	}
	if got := len(h.listCategories(work.ID, nil).Items); got != model.MaxSystemCategories {
		t.Fatalf("应达到 %d 类，实际 %d", model.MaxSystemCategories, got)
	}
	rec := h.do(http.MethodPost, "/api/v1/system-categories", map[string]any{
		"work_id":       work.ID,
		"name":          "第十一类",
		"category_type": model.CategoryTypeNormal,
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_LIMIT")
}

func TestSystemCategoriesDeleteGroup(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("类别整组删除")
	category := h.createCategory(work.ID, "整组", model.CategoryTypeNormal)
	repo := systemRepoForTest(h)

	root := seedNode(t, h, repo, work.ID, category.ID, nil, "主层级", model.ReviewStatusReviewed)
	seedNode(t, h, repo, work.ID, category.ID, &root.ID, "子层级", model.ReviewStatusReviewed)

	rec := h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	result := decodeBody[categoryDeletePayload](t, rec)
	if !result.Deleted || result.UndoToken == "" || result.AffectedNodes != 2 {
		t.Fatalf("删除结果不符：%+v", result)
	}

	var deletedRows int64
	if err := h.tx.Table("system_categories").
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", category.ID).
		Count(&deletedRows).Error; err != nil {
		t.Fatalf("检查类别软删除失败：%v", err)
	}
	if deletedRows != 1 {
		t.Fatalf("类别应带删除三件套，实际 %d", deletedRows)
	}
	var deletedNodes int64
	if err := h.tx.Table("system_nodes").
		Where("category_id = ? AND deleted_at IS NOT NULL AND deleted_root_id = ?", category.ID, category.ID).
		Count(&deletedNodes).Error; err != nil {
		t.Fatalf("检查节点软删除失败：%v", err)
	}
	if deletedNodes != 2 {
		t.Fatalf("节点应随组软删除，实际 %d", deletedNodes)
	}
	var tokens int64
	if err := h.tx.Table("undo_tokens").
		Where("root_type = ? AND root_id = ?", "system_category", category.ID).
		Count(&tokens).Error; err != nil {
		t.Fatalf("检查撤销令牌失败：%v", err)
	}
	if tokens != 1 {
		t.Fatalf("应写入 1 条撤销令牌，实际 %d", tokens)
	}

	rec = h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusNotFound, "NOT_FOUND")
}

func TestSystemCategoriesReferenceConflict(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("引用阻断")
	category := h.createCategory(work.ID, "引用类别", model.CategoryTypeNormal)
	repo := systemRepoForTest(h)
	node := seedNode(t, h, repo, work.ID, category.ID, nil, "金丹", model.ReviewStatusReviewed)
	seedReference(t, h, work.ID, "char-1", category.ID, node.ID)

	rec := h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{
		"is_enabled": false,
	}, map[string]string{"If-Match": `"1"`})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	var items []struct {
		Type     string  `json:"type"`
		EntityID string  `json:"entity_id"`
		Name     *string `json:"name"`
		Field    string  `json:"field"`
		Value    string  `json:"value"`
		InTrash  bool    `json:"in_trash"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &items); err != nil {
		t.Fatalf("解析引用明细失败：%v，body=%s", err, envelope.Error.Details)
	}
	if len(items) != 1 || items[0].EntityID != "char-1" || items[0].Field != "node" || items[0].Value != "金丹" || items[0].Name == nil || *items[0].Name != "char-1" {
		t.Fatalf("引用明细不符：%+v", items)
	}

	rec = h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	// 引用检查与删除共用事务，阻断后类别与节点均不得出现软删除痕迹。
	var categoryDeleted, nodeDeleted int64
	if err := h.tx.Table("system_categories").
		Where("id = ? AND deleted_at IS NOT NULL", category.ID).
		Count(&categoryDeleted).Error; err != nil {
		t.Fatalf("检查类别软删除失败：%v", err)
	}
	if err := h.tx.Table("system_nodes").
		Where("id = ? AND deleted_at IS NOT NULL", node.ID).
		Count(&nodeDeleted).Error; err != nil {
		t.Fatalf("检查节点软删除失败：%v", err)
	}
	if categoryDeleted != 0 || nodeDeleted != 0 {
		t.Fatalf("引用阻断后不应软删除（类别=%d，节点=%d）", categoryDeleted, nodeDeleted)
	}

	if err := h.tx.Exec("DELETE FROM character_system_refs WHERE category_id = ?", category.ID).Error; err != nil {
		t.Fatalf("清理引用失败：%v", err)
	}
	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{
		"is_enabled": false,
	}, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("清理引用后停用应成功（status=%d）：%s", rec.Code, rec.Body.String())
	}
}

func TestSystemCrossWorkIsolation(t *testing.T) {
	h := newHarness(t)
	first := h.createWork("作品甲")
	second := h.createWork("作品乙")

	category := h.createCategory(first.ID, "同名类别", model.CategoryTypeNormal)
	other := h.createCategory(second.ID, "同名类别", model.CategoryTypeNormal)
	node := seedNode(t, h, systemRepoForTest(h), first.ID, category.ID, nil, "金丹", model.ReviewStatusReviewed)

	list := h.listCategories(second.ID, nil)
	if len(list.Items) != 2 {
		t.Fatalf("作品乙应只有内置与自己的类别，实际 %d：%+v", len(list.Items), list.Items)
	}
	for _, item := range list.Items {
		if item.WorkID != second.ID {
			t.Fatalf("列表混入其他作品类别：%+v", item)
		}
	}

	nodes := h.getNodes(category.ID, "")
	if len(nodes.Items) != 1 || nodes.Items[0].ID != node.ID {
		t.Fatalf("节点列表应按类别隔离：%+v", nodes.Items)
	}

	empty := h.getNodes(other.ID, "")
	if len(empty.Items) != 0 {
		t.Fatalf("另一作品的类别不应含节点：%+v", empty.Items)
	}
}

func errorField(t *testing.T, envelope errorEnvelope) string {
	t.Helper()
	var details struct {
		Field string `json:"field"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil {
		t.Fatalf("解析错误字段失败：%v，body=%s", err, envelope.Error.Details)
	}
	return details.Field
}

type referenceItemPayload struct {
	Type     string  `json:"type"`
	EntityID string  `json:"entity_id"`
	Name     *string `json:"name"`
	Field    string  `json:"field"`
	Value    string  `json:"value"`
	InTrash  bool    `json:"in_trash"`
}

func decodeReferenceItems(t *testing.T, envelope errorEnvelope) []referenceItemPayload {
	t.Helper()
	var items []referenceItemPayload
	if err := json.Unmarshal(envelope.Error.Details, &items); err != nil {
		t.Fatalf("解析引用明细失败：%v，body=%s", err, envelope.Error.Details)
	}
	return items
}
