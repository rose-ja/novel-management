package model

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantName string
		wantKey  string
		wantErr  bool
	}{
		{name: "去空白并小写化", input: "  山海  ", wantName: "山海", wantKey: "山海"},
		{name: "英文名称小写化", input: " Sword ", wantName: "Sword", wantKey: "sword"},
		{name: "空名称", input: "   ", wantErr: true},
		{name: "超长名称", input: strings.Repeat("长", MaxNameRunes+1), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, key, err := NormalizeName(tc.input)
			if tc.wantErr {
				var validation *ValidationError
				if !errors.As(err, &validation) {
					t.Fatalf("期望 ValidationError，实际 %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("非预期错误：%v", err)
			}
			if name != tc.wantName || key != tc.wantKey {
				t.Fatalf("期望 %q/%q，实际 %q/%q", tc.wantName, tc.wantKey, name, key)
			}
		})
	}
}

func TestNormalizeDescription(t *testing.T) {
	if got, err := NormalizeDescription(nil); err != nil || got != nil {
		t.Fatalf("nil 应返回 nil：%v %v", got, err)
	}
	empty := "   "
	if got, err := NormalizeDescription(&empty); err != nil || got != nil {
		t.Fatalf("空白应视为未填写：%v %v", got, err)
	}
	text := "  设定说明  "
	got, err := NormalizeDescription(&text)
	if err != nil || got == nil || *got != "设定说明" {
		t.Fatalf("描述应去除首尾空白：%v %v", got, err)
	}
	tooLong := strings.Repeat("描", MaxDescriptionRunes+1)
	if _, err := NormalizeDescription(&tooLong); err == nil {
		t.Fatal("超长描述应返回错误")
	}
}
