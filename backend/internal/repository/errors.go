package repository

import (
	"strings"
)

// IsUniqueViolation 判断错误是否由指定唯一索引触发；缺少 pgx 依赖时用错误文本匹配，
// 单一用户场景下足够稳定。
func IsUniqueViolation(err error, index string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), index)
}
