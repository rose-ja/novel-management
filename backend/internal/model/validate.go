package model

import (
	"strings"
	"unicode/utf8"
)

// 与 /meta 契约保持一致的限制值。
const (
	MaxNameRunes        = 200
	MaxDescriptionRunes = 20000
)

// NormalizeName 去除首尾空白并生成 name_key；空名与超长均为 ValidationError。
func NormalizeName(raw string) (name, nameKey string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", &ValidationError{Field: "name", Message: "作品名称不能为空"}
	}
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return "", "", &ValidationError{Field: "name", Message: "作品名称不能超过 200 个字符"}
	}
	return name, strings.ToLower(name), nil
}

// NormalizeDescription 空字符串视为未填写（nil），并校验长度上限。
func NormalizeDescription(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxDescriptionRunes {
		return nil, &ValidationError{Field: "description", Message: "作品描述不能超过 20000 个字符"}
	}
	return &value, nil
}
