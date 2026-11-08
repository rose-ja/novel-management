package model

import (
	"testing"
)

func TestNormalizeKey(t *testing.T) {
	cases := []struct {
		raw, display, key string
	}{
		{"  炼气  ", "炼气", "炼气"},
		{"Ｓｗｏｒｄ", "Ｓｗｏｒｄ", "sword"},
		{"Sword", "Sword", "sword"},
		{"  金   丹  ", "金   丹", "金 丹"},
	}
	for _, tc := range cases {
		display, key, err := NormalizeKey(tc.raw, "name")
		if err != nil {
			t.Fatalf("NormalizeKey(%q) error: %v", tc.raw, err)
		}
		if display != tc.display || key != tc.key {
			t.Fatalf("NormalizeKey(%q) = (%q, %q), want (%q, %q)", tc.raw, display, key, tc.display, tc.key)
		}
	}
}

func TestNormalizeKeyRejectsEmptyAndLong(t *testing.T) {
	if _, _, err := NormalizeKey("   ", "name"); err == nil {
		t.Fatal("空白名称应返回校验错误")
	}
	long := make([]rune, MaxNameRunes+1)
	for i := range long {
		long[i] = '气'
	}
	if _, _, err := NormalizeKey(string(long), "name"); err == nil {
		t.Fatal("超长名称应返回校验错误")
	}
}

func TestAssembleNodeTreeOrderAndTwoLevels(t *testing.T) {
	parentA := "a"
	parentB := "b"
	nodes := []SystemNode{
		{ID: "b", Name: "乙", SortOrder: 2},
		{ID: "a", Name: "甲", SortOrder: 1},
		{ID: "a2", Name: "甲二", ParentID: &parentA, SortOrder: 2},
		{ID: "a1", Name: "甲一", ParentID: &parentA, SortOrder: 1},
		{ID: "b1", Name: "乙一", ParentID: &parentB, SortOrder: 1},
	}
	roots, err := AssembleNodeTree(nodes)
	if err != nil {
		t.Fatalf("AssembleNodeTree error: %v", err)
	}
	if len(roots) != 2 || roots[0].ID != "a" || roots[1].ID != "b" {
		t.Fatalf("root order = %+v", roots)
	}
	if len(roots[0].Children) != 2 || roots[0].Children[0].ID != "a1" {
		t.Fatalf("children order = %+v", roots[0].Children)
	}

	thirdParent := "a1"
	bad := []SystemNode{
		{ID: "a", SortOrder: 1},
		{ID: "a1", ParentID: &parentA, SortOrder: 1},
		{ID: "a11", ParentID: &thirdParent, SortOrder: 1},
	}
	if _, err := AssembleNodeTree(bad); err == nil {
		t.Fatal("三级节点应返回错误")
	}
}

func TestValidateSiblingOrder(t *testing.T) {
	if err := ValidateSiblingOrder([]string{"a", "b"}, []string{"b", "a"}); err != nil {
		t.Fatalf("合法排序被拒绝: %v", err)
	}
	if err := ValidateSiblingOrder([]string{"a", "b"}, []string{"a"}); err == nil {
		t.Fatal("缺失节点应返回错误")
	}
	if err := ValidateSiblingOrder([]string{"a"}, []string{"a", "a"}); err == nil {
		t.Fatal("重复节点应返回错误")
	}
	if err := ValidateSiblingOrder([]string{"a"}, []string{"a", "b"}); err == nil {
		t.Fatal("非同级节点应返回错误")
	}
}

func TestReviewStatusMapping(t *testing.T) {
	if got := APIReviewStatus(ReviewStatusReviewed); got != ReviewStatusAPIConfirmed {
		t.Fatalf("reviewed 映射 = %q", got)
	}
	if got := APIReviewStatus(ReviewStatusPending); got != ReviewStatusAPIPending {
		t.Fatalf("pending 映射 = %q", got)
	}
	db, err := DBReviewStatus(ReviewStatusAPIConfirmed)
	if err != nil || db != ReviewStatusReviewed {
		t.Fatalf("confirmed 反向映射 = %q, %v", db, err)
	}
	if _, err := DBReviewStatus("unknown"); err == nil {
		t.Fatal("未知 review_status 应返回错误")
	}
}

func TestNormalizeEraText(t *testing.T) {
	empty := "   "
	got, err := NormalizeEraText(&empty, "era_start_text")
	if err != nil || got != nil {
		t.Fatalf("空白纪元文本应转为 nil: %v, %v", got, err)
	}
	value := " 上古 "
	got, err = NormalizeEraText(&value, "era_start_text")
	if err != nil || got == nil || *got != "上古" {
		t.Fatalf("纪元文本 trim 失败: %v, %v", got, err)
	}
}
