package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumsVerifyAndDetectTampering(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "data", "app.dump"), "dump")
	writeFile(t, filepath.Join(root, "manifest.json"), "{}")

	entries, err := ListFiles(root, nil)
	if err != nil {
		t.Fatalf("列举文件失败：%v", err)
	}
	checksumPath := filepath.Join(root, ChecksumFile)
	if err := WriteChecksums(root, checksumPath, entries); err != nil {
		t.Fatalf("写校验和失败：%v", err)
	}
	if err := VerifyChecksums(root, checksumPath); err != nil {
		t.Fatalf("校验应通过：%v", err)
	}

	writeFile(t, filepath.Join(root, "data", "app.dump"), "tampered")
	if err := VerifyChecksums(root, checksumPath); err == nil {
		t.Fatal("篡改后校验应失败")
	}

	writeFile(t, filepath.Join(root, "data", "app.dump"), "dump")
	writeFile(t, filepath.Join(root, "extra.txt"), "extra")
	if err := VerifyChecksums(root, checksumPath); err == nil {
		t.Fatal("未登记文件应导致校验失败")
	}

	if err := os.Remove(filepath.Join(root, "extra.txt")); err != nil {
		t.Fatalf("删除文件失败：%v", err)
	}
	writeFile(t, filepath.Join(root, "manifest.json"), "{\"changed\":true}")
	if err := VerifyChecksums(root, checksumPath); err == nil {
		t.Fatal("内容变化后校验应失败")
	}
}
