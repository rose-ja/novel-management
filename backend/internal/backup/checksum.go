package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ChecksumFile 是固定文件名，校验和不覆盖自身。
const ChecksumFile = "checksums.sha256"

// WriteChecksums 计算 entries 的 sha256 并写入 outPath，格式与 sha256sum 一致。
func WriteChecksums(root, outPath string, entries []Entry) error {
	var builder strings.Builder
	for _, entry := range entries {
		if entry.Path == ChecksumFile {
			continue
		}
		sum, err := FileSHA256(filepath.Join(root, filepath.FromSlash(entry.Path)))
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder, "%s  %s\n", sum, entry.Path)
	}
	return os.WriteFile(outPath, []byte(builder.String()), 0o644)
}

// VerifyChecksums 校验 root 内所有文件与 checksums.sha256 一致，且不存在未列出的文件。
func VerifyChecksums(root, checksumPath string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("读取校验和失败：%w", err)
	}

	expected := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || len(parts[0]) != 64 {
			return fmt.Errorf("校验和格式无效：%q", line)
		}
		rel := strings.TrimPrefix(parts[1], "*")
		if _, err := SafeJoin(root, rel); err != nil {
			return err
		}
		expected[rel] = parts[0]
	}
	if len(expected) == 0 {
		return fmt.Errorf("校验和文件为空")
	}

	actual, err := ListFiles(root, map[string]bool{ChecksumFile: true})
	if err != nil {
		return err
	}
	for _, entry := range actual {
		want, ok := expected[entry.Path]
		if !ok {
			return fmt.Errorf("导出包含未登记文件：%s", entry.Path)
		}
		sum, err := FileSHA256(filepath.Join(root, filepath.FromSlash(entry.Path)))
		if err != nil {
			return err
		}
		if sum != want {
			return fmt.Errorf("校验和不匹配：%s", entry.Path)
		}
		delete(expected, entry.Path)
	}
	for missing := range expected {
		return fmt.Errorf("导出缺少文件：%s", missing)
	}
	return nil
}

// FileSHA256 返回文件内容的十六进制 sha256。
func FileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
