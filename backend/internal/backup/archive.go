package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Entry 描述一个待打包文件：相对 root 的斜杠路径与大小。
type Entry struct {
	Path string
	Size int64
}

// ListFiles 递归收集 root 下文件（相对路径、斜杠分隔），按路径排序；skip 按相对路径排除。
func ListFiles(root string, skip map[string]bool) ([]Entry, error) {
	entries := make([]Entry, 0, 64)
	err := filepath.Walk(root, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if skip[rel] {
			return nil
		}
		entries = append(entries, Entry{Path: rel, Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// WriteZip 按 entries 顺序写入 zip；data/ 下的转储使用 Store 避免重复压缩。
func WriteZip(zipPath, root string, entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(out)

	writeErr := error(nil)
	for _, entry := range entries {
		source, err := OpenInside(root, entry.Path)
		if err != nil {
			writeErr = err
			break
		}
		sourceInfo, err := source.Stat()
		if err != nil {
			_ = source.Close()
			writeErr = err
			break
		}
		header, err := zip.FileInfoHeader(sourceInfo)
		if err != nil {
			_ = source.Close()
			writeErr = err
			break
		}
		header.Name = entry.Path
		header.Method = zip.Deflate
		if strings.HasPrefix(entry.Path, "data/") {
			header.Method = zip.Store
		}
		target, err := writer.CreateHeader(header)
		if err != nil {
			_ = source.Close()
			writeErr = err
			break
		}
		if _, err := io.Copy(target, source); err != nil {
			_ = source.Close()
			writeErr = err
			break
		}
		if err := source.Close(); err != nil {
			writeErr = err
			break
		}
	}

	if writeErr != nil {
		_ = writer.Close()
		_ = out.Close()
		return writeErr
	}
	if err := writer.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// OpenInside 打开 root 内的相对路径，拒绝越界路径。
func OpenInside(root, rel string) (*os.File, error) {
	target, err := SafeJoin(root, rel)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, fmt.Errorf("打开 %s 失败：%w", rel, err)
	}
	return file, nil
}

// ExtractZip 解压到 dest，拒绝绝对路径与 ..；返回解压文件数。
func ExtractZip(zipPath, dest string) (int, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, fmt.Errorf("打开导出包失败：%w", err)
	}
	defer func() { _ = reader.Close() }()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}

	count := 0
	for _, file := range reader.File {
		target, err := SafeJoin(dest, file.Name)
		if err != nil {
			return count, err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return count, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return count, err
		}
		if err := extractFile(file, target); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func extractFile(file *zip.File, target string) error {
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, source); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// SafeJoin 将 zip 内的斜杠路径安全地拼到 root；拒绝绝对路径、盘符与 .. 越界。
func SafeJoin(root, name string) (string, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	clean := path.Clean(normalized)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return "", fmt.Errorf("导出包含非法路径：%s", name)
	}
	if strings.Contains(clean, ":") {
		return "", fmt.Errorf("导出包含非法路径：%s", name)
	}
	target := filepath.Join(root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("导出包含非法路径：%s", name)
	}
	return target, nil
}
