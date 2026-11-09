package service

import (
	"errors"
	"strings"
	"testing"

	"novel-management/backend/internal/model"
)

func TestNormalizeCharacterNamesDedupeAndOrder(t *testing.T) {
	got, err := normalizeCharacterNames([]CharacterNameInput{
		{Kind: "alias", Value: " 小圣 "},
		{Kind: "title", Value: "山海帝君"},
		{Kind: "alias", Value: "小圣"},
	})
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	if len(got) != 2 {
		t.Fatalf("期望去重后 2 条，实际 %d", len(got))
	}
	if got[0].Value != "小圣" || got[0].SortOrder != 0 {
		t.Fatalf("首条别名不正确：%+v", got[0])
	}
	if got[1].Kind != "title" || got[1].SortOrder != 1 {
		t.Fatalf("称号顺序不正确：%+v", got[1])
	}
}

func TestNormalizeCharacterNamesRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name   string
		inputs []CharacterNameInput
		field  string
	}{
		{"非法 kind", []CharacterNameInput{{Kind: "nickname", Value: "小圣"}}, "names"},
		{"空值", []CharacterNameInput{{Kind: "alias", Value: "  "}}, "names"},
		{"超长", []CharacterNameInput{{Kind: "alias", Value: strings.Repeat("龙", 101)}}, "names"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizeCharacterNames(tc.inputs)
			var validation *model.ValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field {
				t.Fatalf("期望字段 %s 的 ValidationError，实际 %v", tc.field, err)
			}
		})
	}

	tooMany := make([]CharacterNameInput, model.MaxCharacterNames+1)
	for i := range tooMany {
		tooMany[i] = CharacterNameInput{Kind: "alias", Value: "别名" + strings.Repeat("a", i%5) + string(rune('一'+i%20))}
	}
	if _, err := normalizeCharacterNames(tooMany); err == nil {
		t.Fatal("超过硬上限应返回错误")
	}
}

func TestNormalizeCharacterRefs(t *testing.T) {
	got, err := normalizeCharacterRefs([]CharacterSystemRefInput{
		{CategoryID: " cat1 ", NodeID: " node1 "},
	})
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	if got[0].CategoryID != "cat1" || got[0].NodeID != "node1" {
		t.Fatalf("未去除首尾空白：%+v", got[0])
	}

	cases := []struct {
		name  string
		input []CharacterSystemRefInput
	}{
		{"重复类别", []CharacterSystemRefInput{
			{CategoryID: "cat1", NodeID: "node1"},
			{CategoryID: "cat1", NodeID: "node2"},
		}},
		{"缺节点", []CharacterSystemRefInput{{CategoryID: "cat1"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := normalizeCharacterRefs(tc.input); err == nil {
				t.Fatal("期望返回校验错误")
			}
		})
	}
}

func TestParseCharacterFilter(t *testing.T) {
	filter, page, pageSize, err := parseCharacterFilter(ListCharactersInput{
		WorkID: " work1 ",
		Query:  " 陆沉 ",
	})
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if filter.WorkID != "work1" || filter.Query != "陆沉" {
		t.Fatalf("未去除空白：%+v", filter)
	}
	if page != 1 || pageSize != 20 {
		t.Fatalf("默认分页应为 1/20，实际 %d/%d", page, pageSize)
	}

	_, _, pageSize, err = parseCharacterFilter(ListCharactersInput{WorkID: "w", PageSize: 500})
	if err != nil || pageSize != 100 {
		t.Fatalf("page_size 应夹取到 100，实际 %d（err=%v）", pageSize, err)
	}

	invalid := []struct {
		name  string
		input ListCharactersInput
		field string
	}{
		{"缺作品", ListCharactersInput{}, "work_id"},
		{"地点未开放", ListCharactersInput{WorkID: "w", LocationID: "loc"}, "location_id"},
		{"非法排序", ListCharactersInput{WorkID: "w", Sort: "created_at"}, "sort"},
		{"非法校对态", ListCharactersInput{WorkID: "w", ReviewStatus: "draft"}, "review_status"},
		{"搜索词超长", ListCharactersInput{WorkID: "w", Query: strings.Repeat("龙", 101)}, "q"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseCharacterFilter(tc.input)
			var validation *model.ValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field {
				t.Fatalf("期望字段 %s 的 ValidationError，实际 %v", tc.field, err)
			}
		})
	}
}
