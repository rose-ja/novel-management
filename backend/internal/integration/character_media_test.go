//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mediaViewPayload struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	ByteSize int64  `json:"byte_size"`
}

type chroniclePayload struct {
	ID        string  `json:"id"`
	OwnerType string  `json:"owner_type"`
	OwnerID   string  `json:"owner_id"`
	Text      string  `json:"text"`
	TimeLabel *string `json:"time_label"`
	SortOrder int     `json:"sort_order"`
	Version   int     `json:"version"`
}

func makePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("生成 PNG 失败：%v", err)
	}
	return buf.Bytes()
}

func makeJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 80, G: uint8(x % 255), B: uint8(y % 255), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("生成 JPEG 失败：%v", err)
	}
	return buf.Bytes()
}

func (h *harness) uploadImage(characterID, filename string, data []byte) *httptest.ResponseRecorder {
	h.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		h.t.Fatalf("构造 multipart 失败：%v", err)
	}
	if _, err := part.Write(data); err != nil {
		h.t.Fatalf("写入 multipart 失败：%v", err)
	}
	if err := writer.Close(); err != nil {
		h.t.Fatalf("关闭 multipart 失败：%v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:17860/api/v1/characters/"+characterID+"/image", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rec := httptest.NewRecorder()
	h.engine.ServeHTTP(rec, req)
	return rec
}

func TestCharacterImageUploadAndServe(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("图片作品")
	character := h.createCharacter(work.ID, "陆沉", nil, nil)

	uploaded := h.uploadImage(character.ID, "avatar.png", makePNG(t, 800, 600))
	if uploaded.Code != http.StatusCreated {
		t.Fatalf("上传失败（status=%d）：%s", uploaded.Code, uploaded.Body.String())
	}
	first := decodeBody[mediaViewPayload](t, uploaded)
	if first.AssetID == "" || first.URL == "" || first.ThumbURL == "" {
		t.Fatalf("上传响应缺少字段：%+v", first)
	}
	if first.Width != 800 || first.Height != 600 {
		t.Fatalf("尺寸应为 800x600，实际 %dx%d", first.Width, first.Height)
	}
	originalPath := filepath.Join(h.dataDir, "images", work.ID, first.AssetID+".png")
	thumbPath := filepath.Join(h.dataDir, "images", work.ID, first.AssetID+"_thumb.png")
	if _, err := os.Stat(originalPath); err != nil {
		t.Fatalf("原图未落盘：%v", err)
	}
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		t.Fatalf("缩略图未落盘：%v", err)
	}
	thumbConfig, err := png.DecodeConfig(thumbFile)
	_ = thumbFile.Close()
	if err != nil {
		t.Fatalf("解析缩略图失败：%v", err)
	}
	if thumbConfig.Width != 320 || thumbConfig.Height != 240 {
		t.Fatalf("缩略图应为 320x240，实际 %dx%d", thumbConfig.Width, thumbConfig.Height)
	}

	served := h.do(http.MethodGet, first.URL, nil, nil)
	if served.Code != http.StatusOK {
		t.Fatalf("读取原图失败（status=%d）", served.Code)
	}
	if got := served.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type 应为 image/png，实际 %q", got)
	}
	etag := served.Header().Get("ETag")
	if etag == "" {
		t.Fatal("缺少 ETag")
	}
	notModified := h.do(http.MethodGet, first.URL, nil, map[string]string{"If-None-Match": etag})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("ETag 命中应返回 304，实际 %d", notModified.Code)
	}
	thumbServed := h.do(http.MethodGet, first.ThumbURL, nil, nil)
	if thumbServed.Code != http.StatusOK {
		t.Fatalf("读取缩略图失败（status=%d）", thumbServed.Code)
	}

	var detail struct {
		Image *mediaViewPayload `json:"image"`
	}
	detailRec := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("读取详情失败（status=%d）", detailRec.Code)
	}
	if err := decodeJSON(t, detailRec, &detail); err != nil {
		t.Fatalf("解析详情失败：%v", err)
	}
	if detail.Image == nil || detail.Image.AssetID != first.AssetID {
		t.Fatalf("详情未返回主图：%+v", detail.Image)
	}

	var list struct {
		Items []struct {
			ImageThumbURL *string `json:"image_thumb_url"`
		} `json:"items"`
	}
	listRec := h.do(http.MethodGet, "/api/v1/characters?work_id="+work.ID, nil, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("读取列表失败（status=%d）", listRec.Code)
	}
	if err := decodeJSON(t, listRec, &list); err != nil {
		t.Fatalf("解析列表失败：%v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ImageThumbURL == nil {
		t.Fatalf("列表应返回缩略图：%+v", list.Items)
	}

	replaced := h.uploadImage(character.ID, "avatar.jpg", makeJPEG(t, 100, 100))
	if replaced.Code != http.StatusCreated {
		t.Fatalf("替换失败（status=%d）：%s", replaced.Code, replaced.Body.String())
	}
	second := decodeBody[mediaViewPayload](t, replaced)
	if second.AssetID == first.AssetID {
		t.Fatal("替换后资产 ID 不应相同")
	}
	if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
		t.Fatalf("替换后旧图应删除，实际 err=%v", err)
	}
	var oldDeleted int64
	if err := h.tx.Raw("SELECT count(*) FROM media_assets WHERE id = ? AND deleted_at IS NOT NULL", first.AssetID).Scan(&oldDeleted).Error; err != nil {
		t.Fatalf("查询旧资产失败：%v", err)
	}
	if oldDeleted != 1 {
		t.Fatalf("旧资产应软删除，实际 %d", oldDeleted)
	}

	removed := h.do(http.MethodDelete, "/api/v1/characters/"+character.ID+"/image", nil, nil)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("移除主图失败（status=%d）：%s", removed.Code, removed.Body.String())
	}
	if gone := h.do(http.MethodGet, second.URL, nil, nil); gone.Code != http.StatusNotFound {
		t.Fatalf("移除后读取应 404，实际 %d", gone.Code)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "images", work.ID, second.AssetID+".jpg")); !os.IsNotExist(err) {
		t.Fatalf("移除后文件应删除，实际 err=%v", err)
	}
	if again := h.do(http.MethodDelete, "/api/v1/characters/"+character.ID+"/image", nil, nil); again.Code != http.StatusNotFound {
		t.Fatalf("重复移除应 404，实际 %d", again.Code)
	}
}

func TestCharacterImageValidation(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("图片校验")
	character := h.createCharacter(work.ID, "校验人物", nil, nil)

	cases := []struct {
		name     string
		filename string
		data     []byte
		code     string
	}{
		{name: "文本伪装", filename: "fake.png", data: []byte("not an image"), code: "UNSUPPORTED_FILE_TYPE"},
		{name: "PNG 头但数据损坏", filename: "bad.png", data: []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2}, code: "UNSUPPORTED_FILE_TYPE"},
		{name: "GIF", filename: "anim.gif", data: []byte("GIF89a----"), code: "UNSUPPORTED_FILE_TYPE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := h.uploadImage(character.ID, tc.filename, tc.data)
			assertErrorCode(t, rec, http.StatusUnprocessableEntity, tc.code)
		})
	}

	large := make([]byte, 11*1024*1024)
	copy(large, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	assertErrorCode(t, h.uploadImage(character.ID, "big.png", large), http.StatusUnprocessableEntity, "FILE_TOO_LARGE")

	var rows int64
	if err := h.tx.Table("media_assets").Count(&rows).Error; err != nil {
		t.Fatalf("统计资产失败：%v", err)
	}
	if rows != 0 {
		t.Fatalf("非法上传不应写入资产，实际 %d 行", rows)
	}
}

func TestChronicleCRUDAndUndo(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("大事记作品")
	character := h.createCharacter(work.ID, "记录者", nil, nil)

	first := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "character", "owner_id": character.ID, "text": "初入宗门", "time_label": "元启三年",
	}, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("创建大事记失败（status=%d）：%s", first.Code, first.Body.String())
	}
	firstPayload := decodeBody[chroniclePayload](t, first)
	second := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "character", "owner_id": character.ID, "text": "突破金丹",
	}, nil)
	if second.Code != http.StatusCreated {
		t.Fatalf("创建第二条失败（status=%d）：%s", second.Code, second.Body.String())
	}
	secondPayload := decodeBody[chroniclePayload](t, second)
	if firstPayload.SortOrder != 0 || secondPayload.SortOrder != 1 {
		t.Fatalf("排序应为 0/1，实际 %d/%d", firstPayload.SortOrder, secondPayload.SortOrder)
	}

	var list struct {
		Items []chroniclePayload `json:"items"`
	}
	listRec := h.do(http.MethodGet, "/api/v1/chronicles?owner_type=character&owner_id="+character.ID, nil, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("列表失败（status=%d）", listRec.Code)
	}
	if err := decodeJSON(t, listRec, &list); err != nil {
		t.Fatalf("解析列表失败：%v", err)
	}
	if len(list.Items) != 2 || list.Items[0].ID != firstPayload.ID {
		t.Fatalf("列表顺序不正确：%+v", list.Items)
	}

	conflict := h.do(http.MethodPatch, "/api/v1/chronicles/"+firstPayload.ID, map[string]any{
		"text": "改名",
	}, map[string]string{"If-Match": `"999"`})
	assertErrorCode(t, conflict, http.StatusConflict, "VERSION_CONFLICT")

	updated := h.do(http.MethodPatch, "/api/v1/chronicles/"+firstPayload.ID, map[string]any{
		"text": "初入宗门（修订）", "time_label": nil,
	}, map[string]string{"If-Match": fmt.Sprintf(`"%d"`, firstPayload.Version)})
	if updated.Code != http.StatusOK {
		t.Fatalf("更新失败（status=%d）：%s", updated.Code, updated.Body.String())
	}
	updatedPayload := decodeBody[chroniclePayload](t, updated)
	if updatedPayload.Text != "初入宗门（修订）" || updatedPayload.TimeLabel != nil {
		t.Fatalf("更新结果不正确：%+v", updatedPayload)
	}
	if updatedPayload.Version != firstPayload.Version+1 {
		t.Fatalf("版本应递增，实际 %d", updatedPayload.Version)
	}

	tooLong := strings.Repeat("字", 2001)
	invalid := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "character", "owner_id": character.ID, "text": tooLong,
	}, nil)
	envelope := assertErrorCode(t, invalid, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := decodeFieldDetails(t, envelope); field != "text" {
		t.Fatalf("错误字段应为 text，实际 %q", field)
	}

	deleted := h.do(http.MethodDelete, "/api/v1/chronicles/"+firstPayload.ID, nil, nil)
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	var deletePayload struct {
		UndoToken string `json:"undo_token"`
	}
	if err := decodeJSON(t, deleted, &deletePayload); err != nil {
		t.Fatalf("解析删除结果失败：%v", err)
	}
	if deletePayload.UndoToken == "" {
		t.Fatal("删除应返回撤销令牌")
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deletePayload.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	listRec = h.do(http.MethodGet, "/api/v1/chronicles?owner_type=character&owner_id="+character.ID, nil, nil)
	if err := decodeJSON(t, listRec, &list); err != nil {
		t.Fatalf("撤销后解析列表失败：%v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("撤销后应恢复 2 条，实际 %d", len(list.Items))
	}

	unknown := h.do(http.MethodGet, "/api/v1/chronicles?owner_type=character&owner_id=missing", nil, nil)
	assertErrorCode(t, unknown, http.StatusNotFound, "NOT_FOUND")
	unsupported := h.do(http.MethodGet, "/api/v1/chronicles?owner_type=faction&owner_id="+character.ID, nil, nil)
	assertErrorCode(t, unsupported, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	reserved := h.do(http.MethodGet, "/api/v1/trash?root_type=chronicle&page=1&page_size=10", nil, nil)
	assertErrorCode(t, reserved, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

func TestWorkLifecycleCoversMediaAndChronicles(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("联动作品组")
	character := h.createCharacter(work.ID, "组内人物", nil, nil)
	uploaded := h.uploadImage(character.ID, "avatar.png", makePNG(t, 120, 120))
	if uploaded.Code != http.StatusCreated {
		t.Fatalf("上传失败（status=%d）：%s", uploaded.Code, uploaded.Body.String())
	}
	view := decodeBody[mediaViewPayload](t, uploaded)
	created := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "character", "owner_id": character.ID, "text": "作品组大事记",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建大事记失败（status=%d）：%s", created.Code, created.Body.String())
	}

	deleted := h.deleteWork(work.ID, work.Name, "1")
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	deletion := decodeBody[deletePayload](t, deleted)
	if served := h.do(http.MethodGet, view.URL, nil, nil); served.Code != http.StatusNotFound {
		t.Fatalf("作品删除后图片应 404，实际 %d", served.Code)
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deletion.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销作品删除失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if served := h.do(http.MethodGet, view.URL, nil, nil); served.Code != http.StatusOK {
		t.Fatalf("恢复作品后图片应可读，实际 %d", served.Code)
	}
	var activeChronicles int64
	if err := h.tx.Table("chronicles").Where("work_id = ? AND deleted_at IS NULL", work.ID).Count(&activeChronicles).Error; err != nil {
		t.Fatalf("统计大事记失败：%v", err)
	}
	if activeChronicles != 1 {
		t.Fatalf("恢复作品后大事记应回来，实际 %d", activeChronicles)
	}

	current := h.getWork(work.ID)
	deletedAgain := h.deleteWork(work.ID, work.Name, fmt.Sprintf("%d", current.Version))
	if deletedAgain.Code != http.StatusOK {
		t.Fatalf("再次删除作品失败（status=%d）：%s", deletedAgain.Code, deletedAgain.Body.String())
	}
	purged := h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除作品失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	var mediaRows, chronicleRows int64
	if err := h.tx.Table("media_assets").Where("work_id = ?", work.ID).Count(&mediaRows).Error; err != nil {
		t.Fatalf("统计媒体失败：%v", err)
	}
	if err := h.tx.Table("chronicles").Where("work_id = ?", work.ID).Count(&chronicleRows).Error; err != nil {
		t.Fatalf("统计大事记失败：%v", err)
	}
	if mediaRows != 0 || chronicleRows != 0 {
		t.Fatalf("永久删除作品后应无残留，media=%d chronicles=%d", mediaRows, chronicleRows)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "images", work.ID, view.AssetID+".png")); !os.IsNotExist(err) {
		t.Fatalf("永久删除作品后图片文件应清理，实际 err=%v", err)
	}
}

func TestCharacterMediaAndChroniclesLifecycle(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("联动作品")
	character := h.createCharacter(work.ID, "联动人物", nil, nil)

	uploaded := h.uploadImage(character.ID, "avatar.png", makePNG(t, 200, 200))
	if uploaded.Code != http.StatusCreated {
		t.Fatalf("上传失败（status=%d）：%s", uploaded.Code, uploaded.Body.String())
	}
	view := decodeBody[mediaViewPayload](t, uploaded)
	created := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "character", "owner_id": character.ID, "text": "随根删除",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建大事记失败（status=%d）：%s", created.Code, created.Body.String())
	}

	deleted := h.do(http.MethodDelete, "/api/v1/characters/"+character.ID, nil, map[string]string{
		"If-Match": fmt.Sprintf(`"%d"`, character.Version),
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除人物失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	deletion := decodeBody[deletePayload](t, deleted)
	if served := h.do(http.MethodGet, view.URL, nil, nil); served.Code != http.StatusNotFound {
		t.Fatalf("删除人物后图片应 404，实际 %d", served.Code)
	}
	var activeChronicles int64
	if err := h.tx.Table("chronicles").Where("owner_id = ? AND deleted_at IS NULL", character.ID).Count(&activeChronicles).Error; err != nil {
		t.Fatalf("统计大事记失败：%v", err)
	}
	if activeChronicles != 0 {
		t.Fatalf("删除人物后大事记应隐藏，实际 %d", activeChronicles)
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deletion.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销删除失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if served := h.do(http.MethodGet, view.URL, nil, nil); served.Code != http.StatusOK {
		t.Fatalf("恢复人物后图片应可读，实际 %d", served.Code)
	}
	if err := h.tx.Table("chronicles").Where("owner_id = ? AND deleted_at IS NULL", character.ID).Count(&activeChronicles).Error; err != nil {
		t.Fatalf("恢复后统计大事记失败：%v", err)
	}
	if activeChronicles != 1 {
		t.Fatalf("恢复人物后大事记应回来，实际 %d", activeChronicles)
	}

	deletedAgain := h.do(http.MethodDelete, "/api/v1/characters/"+character.ID, nil, map[string]string{
		"If-Match": `"3"`,
	})
	if deletedAgain.Code != http.StatusOK {
		t.Fatalf("再次删除失败（status=%d）：%s", deletedAgain.Code, deletedAgain.Body.String())
	}
	purged := h.do(http.MethodDelete, "/api/v1/trash/character/"+character.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	var mediaRows, chronicleRows int64
	if err := h.tx.Table("media_assets").Where("entity_id = ?", character.ID).Count(&mediaRows).Error; err != nil {
		t.Fatalf("统计媒体失败：%v", err)
	}
	if err := h.tx.Table("chronicles").Where("owner_id = ?", character.ID).Count(&chronicleRows).Error; err != nil {
		t.Fatalf("统计大事记失败：%v", err)
	}
	if mediaRows != 0 || chronicleRows != 0 {
		t.Fatalf("永久删除后应无残留，media=%d chronicles=%d", mediaRows, chronicleRows)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "images", work.ID, view.AssetID+".png")); !os.IsNotExist(err) {
		t.Fatalf("永久删除后图片文件应清理，实际 err=%v", err)
	}
}
