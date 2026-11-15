//go:build linux

package backup

import "syscall"

// AvailableBytes 返回目录所在文件系统的可用字节；0 表示未知。
func AvailableBytes(dir string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
