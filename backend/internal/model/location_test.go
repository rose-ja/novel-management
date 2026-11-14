package model

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeLocationName(t *testing.T) {
	name, key, err := NormalizeLocationName("  东荒  ")
	if err != nil || name != "东荒" || key != "东荒" {
		t.Fatalf("名称规范化不正确：%q %q %v", name, key, err)
	}
	name, key, err = NormalizeLocationName("QingQiu")
	if err != nil || name != "QingQiu" || key != "qingqiu" {
		t.Fatalf("name_key 应小写：%q %q %v", name, key, err)
	}

	if _, _, err := NormalizeLocationName("   "); err == nil {
		t.Fatal("空名称应报错")
	} else {
		var validation *ValidationError
		if !errors.As(err, &validation) || validation.Field != "name" {
			t.Fatalf("空名称错误字段不正确：%v", err)
		}
	}
	if _, _, err := NormalizeLocationName(strings.Repeat("龙", MaxLocationName+1)); err == nil {
		t.Fatal("超长名称应报错")
	}
}

func TestNormalizeLocationOptionalFields(t *testing.T) {
	if value, err := NormalizeLocationType(nil); err != nil || value != nil {
		t.Fatalf("nil 类型应保持 nil：%v %v", value, err)
	}
	if value, err := NormalizeLocationType(strPtr("   ")); err != nil || value != nil {
		t.Fatalf("空白类型应转为 nil：%v %v", value, err)
	}
	if value, err := NormalizeLocationType(strPtr(" 星域 ")); err != nil || value == nil || *value != "星域" {
		t.Fatalf("类型应去空白：%v %v", value, err)
	}
	if _, err := NormalizeLocationType(strPtr(strings.Repeat("星", MaxLocationType+1))); err == nil {
		t.Fatal("超长类型应报错")
	}

	if value, err := NormalizeLocationDescription(strPtr("")); err != nil || value != nil {
		t.Fatalf("空简介应转为 nil：%v %v", value, err)
	}
	if _, err := NormalizeLocationDescription(strPtr(strings.Repeat("龙", MaxLocationDescription+1))); err == nil {
		t.Fatal("超长简介应报错")
	}
}

func TestNormalizeRelationNote(t *testing.T) {
	if value, err := NormalizeRelationNote(strPtr("  ")); err != nil || value != nil {
		t.Fatalf("空备注应转为 nil：%v %v", value, err)
	}
	if value, err := NormalizeRelationNote(strPtr(" 出生地 ")); err != nil || value == nil || *value != "出生地" {
		t.Fatalf("备注应去空白：%v %v", value, err)
	}
	if _, err := NormalizeRelationNote(strPtr(strings.Repeat("龙", MaxRelationNote+1))); err == nil {
		t.Fatal("超长备注应报错")
	}
}
