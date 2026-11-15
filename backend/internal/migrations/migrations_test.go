package migrations

import (
	"strconv"
	"strings"
	"testing"
)

// LatestVersion 必须与嵌入的 SQL 文件最大版本号一致，避免新增迁移后忘记更新兼容校验。
func TestLatestVersionMatchesEmbeddedFiles(t *testing.T) {
	entries, err := files.ReadDir(".")
	if err != nil {
		t.Fatalf("读取嵌入迁移失败：%v", err)
	}

	maxVersion := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			continue
		}
		version, err := strconv.Atoi(prefix)
		if err != nil {
			t.Fatalf("迁移文件名版本号无效：%s", name)
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	if maxVersion != LatestVersion {
		t.Fatalf("LatestVersion=%d，嵌入迁移最大版本=%d", LatestVersion, maxVersion)
	}
}
