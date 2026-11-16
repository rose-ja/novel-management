package imports

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"

	"novel-management/backend/internal/backup"
)

// ErrPathTraversal 表示 zip 条目路径越界。
var ErrPathTraversal = errors.New("压缩包含越界路径")

// InventoryZip 读取图片 zip 的条目、大小与文件头 MIME；不解压全部内容。
func InventoryZip(zipPath string) (*Inventory, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("图片 zip 无法读取：%w", err)
	}
	defer func() { _ = reader.Close() }()

	if len(reader.File) > backup.MaxExtractEntries {
		return nil, fmt.Errorf("图片 zip 条目过多")
	}

	inventory := &Inventory{Files: map[string]ImageFile{}}
	var total int64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("图片 zip 不允许符号链接")
		}
		name, err := NormalizeImagePath(file.Name)
		if err != nil {
			return nil, fmt.Errorf("%w：%s", ErrPathTraversal, file.Name)
		}
		if file.UncompressedSize64 > uint64(backup.MaxExtractBytes)-uint64(total) {
			return nil, fmt.Errorf("图片 zip 解压后超过限制")
		}
		total += int64(file.UncompressedSize64)

		header, err := readHeader(file, 512)
		if err != nil {
			return nil, fmt.Errorf("读取图片 %s 失败：%w", name, err)
		}
		inventory.Files[name] = ImageFile{
			Path: name,
			Size: int64(file.UncompressedSize64),
			MIME: DetectImageMIME(header),
		}
	}
	return inventory, nil
}

func readHeader(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(io.LimitReader(reader, limit))
}
