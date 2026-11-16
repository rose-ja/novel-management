package imports

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTemplateHasNoIssues(t *testing.T) {
	spec, issues := Parse(Markdown())
	if len(issues) != 0 {
		t.Fatalf("模板解析不应有错误：%+v", issues)
	}
	if spec.TemplateVersion != TemplateVersion {
		t.Fatalf("模板版本=%q", spec.TemplateVersion)
	}
	if spec.Work.Name != "山海志异" {
		t.Fatalf("作品名=%q", spec.Work.Name)
	}
	if len(spec.Characters) != 2 || len(spec.Factions) != 1 || len(spec.Locations) != 1 {
		t.Fatalf("解析计数不符：%+v", spec)
	}
	if len(spec.Nodes) != 5 || len(spec.Categories) != 2 {
		t.Fatalf("体系解析不符：categories=%d nodes=%d", len(spec.Categories), len(spec.Nodes))
	}
	if got := spec.Characters[0].SystemRefs; len(got) != 0 {
		t.Fatalf("体系字段应在校验阶段解析：%+v", got)
	}
	if spec.Characters[0].Extras["境界"] != "炼气 > 炼气一层" {
		t.Fatalf("境界字段=%q", spec.Characters[0].Extras["境界"])
	}
}

func TestParseRejectsBadVersionAndUnknownSection(t *testing.T) {
	input := "模板版本：9.9\n\n## 未知区块\n名称：x\n"
	_, issues := Parse([]byte(input))
	if !hasCode(issues, CodeTemplateVersion) {
		t.Fatalf("缺少版本错误：%+v", issues)
	}
	if !hasCode(issues, CodeUnknownSection) {
		t.Fatalf("缺少未知区块错误：%+v", issues)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	input := "模板版本：1.0\n\n## 势力\n### 甲\n未知：x\n"
	_, issues := Parse([]byte(input))
	if !hasCode(issues, CodeUnknownField) {
		t.Fatalf("缺少未知字段错误：%+v", issues)
	}
}

func TestValidateExample(t *testing.T) {
	spec, issues := Parse(ExampleMarkdown())
	if len(issues) != 0 {
		t.Fatalf("示例解析失败：%+v", issues)
	}
	zipPath := filepath.Join(t.TempDir(), "example.zip")
	data, err := ExampleZip()
	if err != nil {
		t.Fatalf("生成示例 zip 失败：%v", err)
	}
	if err := os.WriteFile(zipPath, data, 0o644); err != nil {
		t.Fatalf("写入示例 zip 失败：%v", err)
	}
	inventory, err := InventoryZip(zipPath)
	if err != nil {
		t.Fatalf("读取示例 zip 失败：%v", err)
	}
	result := Validate(spec, inventory)
	if len(result.Errors) != 0 {
		t.Fatalf("示例校验失败：%+v", result.Errors)
	}
	if result.Counts.Characters != 1 || result.Counts.Images != 1 {
		t.Fatalf("计数不符：%+v", result.Counts)
	}
	if result.Counts.SystemCategories != 2 || result.Counts.SystemNodes != 3 {
		t.Fatalf("体系计数不符：%+v", result.Counts)
	}
}

func TestValidateDetectsMissingAndExtraImages(t *testing.T) {
	input := strings.Replace(string(ExampleMarkdown()), "images/example.png", "images/missing.png", 1)
	spec, issues := Parse([]byte(input))
	if len(issues) != 0 {
		t.Fatalf("解析失败：%+v", issues)
	}
	inventory := &Inventory{Files: map[string]ImageFile{
		"images/example.png": {Path: "images/example.png", Size: 10, MIME: "image/png"},
	}}
	result := Validate(spec, inventory)
	if !hasCode(result.Errors, CodeImageMissing) || !hasCode(result.Errors, CodeImageExtra) {
		t.Fatalf("图片错误缺失：%+v", result.Errors)
	}
}

func TestValidateDetectsBrokenReferenceAndDuplicate(t *testing.T) {
	input := strings.Replace(string(ExampleMarkdown()), "势力：示例宗门", "势力：不存在", 1)
	input = strings.Replace(input, "\n## 榜单", "\n### 示例人物\n简介：重复\n\n## 榜单", 1)
	spec, issues := Parse([]byte(input))
	if len(issues) != 0 {
		t.Fatalf("解析失败：%+v", issues)
	}
	result := Validate(spec, nil)
	if !hasCode(result.Errors, CodeReferenceMissing) {
		t.Fatalf("缺少引用错误：%+v", result.Errors)
	}
	if !hasCode(result.Errors, CodeDuplicate) {
		t.Fatalf("缺少重复错误：%+v", result.Errors)
	}
}

func TestValidateNodeParentMustExist(t *testing.T) {
	input := strings.Replace(string(Markdown()), "[境界] 炼气 > 炼气一层\n", "", 1)
	spec, issues := Parse([]byte(input))
	if len(issues) != 0 {
		t.Fatalf("解析失败：%+v", issues)
	}
	result := Validate(spec, nil)
	if !hasCode(result.Errors, CodeReferenceMissing) {
		t.Fatalf("缺少节点引用错误：%+v", result.Errors)
	}
}

func TestInventoryRejectsTraversal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	if err := writeZip(path, map[string]string{"../evil.png": "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := InventoryZip(path); err == nil {
		t.Fatal("应拒绝越界路径")
	}
}

func hasCode(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func writeZip(path string, entries map[string]string) error {
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, value := range entries {
		file, err := writer.Create(name)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte(value)); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
