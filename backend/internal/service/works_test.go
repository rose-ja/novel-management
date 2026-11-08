package service

import "testing"

func TestParseIfMatch(t *testing.T) {
	cases := []struct {
		raw    string
		want   int
		wantOK bool
	}{
		{raw: `"3"`, want: 3, wantOK: true},
		{raw: "3", want: 3, wantOK: true},
		{raw: ` "7" `, want: 7, wantOK: true},
		{raw: "", wantOK: false},
		{raw: `"0"`, wantOK: false},
		{raw: "abc", wantOK: false},
	}
	for _, tc := range cases {
		got, ok := ParseIfMatch(tc.raw)
		if ok != tc.wantOK || (ok && got != tc.want) {
			t.Fatalf("ParseIfMatch(%q) = %d/%v，期望 %d/%v", tc.raw, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestNormalizePage(t *testing.T) {
	page, size := normalizePage(0, 0)
	if page != 1 || size != DefaultPageSize {
		t.Fatalf("默认分页不符：%d/%d", page, size)
	}
	page, size = normalizePage(2, MaxPageSize+50)
	if page != 2 || size != MaxPageSize {
		t.Fatalf("上限裁剪不符：%d/%d", page, size)
	}
}
