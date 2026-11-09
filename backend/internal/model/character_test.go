package model

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCharacterName(t *testing.T) {
	name, key, err := NormalizeCharacterName("  陆沉  ")
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	if name != "陆沉" || key != "陆沉" {
		t.Fatalf("名称规范化不正确：name=%q key=%q", name, key)
	}

	if _, _, err := NormalizeCharacterName("   "); err == nil {
		t.Fatal("空名称应返回错误")
	}
	var validation *ValidationError
	_, _, err = NormalizeCharacterName(strings.Repeat("龙", MaxNameRunes+1))
	if !errors.As(err, &validation) || validation.Field != "name" {
		t.Fatalf("超长名称应返回 name 字段错误，实际 %v", err)
	}
}

func TestNormalizeCharacterNameUsesLowerTrimOnly(t *testing.T) {
	// 设计 D6：人物名称不做 NFKC/空白折叠，仅 lower(trim)。
	_, key, err := NormalizeCharacterName("Sword")
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	if key != "sword" {
		t.Fatalf("大小写应归并为 sword，实际 %q", key)
	}
	_, key, err = NormalizeCharacterName("ＡＢ")
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	if key != "ａｂ" {
		t.Fatalf("不应做全半角归并，实际 %q", key)
	}
}

func TestNormalizeCharacterDescription(t *testing.T) {
	if value, err := NormalizeCharacterDescription(nil); err != nil || value != nil {
		t.Fatalf("nil 简介应保持 nil：%v/%v", value, err)
	}
	if value, err := NormalizeCharacterDescription(strPtr("  ")); err != nil || value != nil {
		t.Fatalf("空白简介应转为 nil：%v/%v", value, err)
	}
	if _, err := NormalizeCharacterDescription(strPtr(strings.Repeat("龙", MaxDescriptionRunes+1))); err == nil {
		t.Fatal("超长简介应返回错误")
	}
}

func strPtr(value string) *string { return &value }
