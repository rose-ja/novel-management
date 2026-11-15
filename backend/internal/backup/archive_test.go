package backup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("创建目录失败：%v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入文件失败：%v", err)
	}
}

func TestWriteAndExtractZipRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "data", "app.dump"), "dump-bytes")
	writeFile(t, filepath.Join(root, "images", "a.jpg"), "image-bytes")
	writeFile(t, filepath.Join(root, "manifest.json"), `{"export_version":1}`)

	entries, err := ListFiles(root, nil)
	if err != nil {
		t.Fatalf("列举文件失败：%v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("文件数应为 3，实际 %d", len(entries))
	}
	if entries[0].Path != "data/app.dump" || entries[1].Path != "images/a.jpg" || entries[2].Path != "manifest.json" {
		t.Fatalf("文件排序不符：%+v", entries)
	}

	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	if err := WriteZip(zipPath, root, entries); err != nil {
		t.Fatalf("写 zip 失败：%v", err)
	}

	dest := t.TempDir()
	count, err := ExtractZip(zipPath, dest)
	if err != nil {
		t.Fatalf("解压失败：%v", err)
	}
	if count != 3 {
		t.Fatalf("解压文件数应为 3，实际 %d", count)
	}
	data, err := os.ReadFile(filepath.Join(dest, "data", "app.dump"))
	if err != nil || string(data) != "dump-bytes" {
		t.Fatalf("解压内容不符：%q，%v", data, err)
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("../evil.txt")
	if err != nil {
		t.Fatalf("创建 zip 条目失败：%v", err)
	}
	if _, err := file.Write([]byte("evil")); err != nil {
		t.Fatalf("写 zip 条目失败：%v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 zip 失败：%v", err)
	}

	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	if err := os.WriteFile(zipPath, buffer.Bytes(), 0o644); err != nil {
		t.Fatalf("写入 zip 失败：%v", err)
	}
	if _, err := ExtractZip(zipPath, t.TempDir()); err == nil {
		t.Fatal("越界路径应被拒绝")
	}
}

func TestSafeJoinRejectsAbsoluteAndDrivePaths(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"/etc/passwd", "C:/windows/win.ini", "a/../../b", "..", "."} {
		if _, err := SafeJoin(root, name); err == nil {
			t.Fatalf("路径 %q 应被拒绝", name)
		}
	}
	if _, err := SafeJoin(root, "images/a.jpg"); err != nil {
		t.Fatalf("合法路径不应失败：%v", err)
	}
}
