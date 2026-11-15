//go:build !linux

package backup

// AvailableBytes 在非 Linux 平台不执行磁盘预检，返回 0 表示未知。
func AvailableBytes(string) (uint64, error) {
	return 0, nil
}
