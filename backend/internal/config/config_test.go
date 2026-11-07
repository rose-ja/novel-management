package config

import (
	"os"
	"strings"
	"testing"
)

// unsetEnv 清空变量并在测试结束后恢复，避免受本机环境影响。
func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if old, ok := os.LookupEnv(key); ok {
			os.Unsetenv(key)
			t.Cleanup(func() { os.Setenv(key, old) })
		}
	}
}

func TestLoadUsesDefaultsAndDerivesAllowedHosts(t *testing.T) {
	unsetEnv(t,
		"APP_HOST", "APP_PORT", "ALLOWED_HOSTS", "DATA_DIR", "REDIS_ADDR",
		"REDIS_PASSWORD", "CACHE_TTL_SEC", "MAX_UPLOAD_MB", "SNAPSHOT_KEEP",
		"IMPORT_TTL_MIN", "IMPORT_TIMEOUT_MIN", "LOG_LEVEL", "TZ", "APP_VERSION",
	)
	t.Setenv("DATABASE_URL", "postgres://novel:secret@localhost:5432/novel?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回错误：%v", err)
	}
	if cfg.Host != DefaultHost || cfg.Port != DefaultPort {
		t.Fatalf("默认监听地址不符：%s", cfg.Addr())
	}
	if cfg.DataDir != "/data" || cfg.SnapshotKeep != 3 || cfg.CacheTTLSec != 60 {
		t.Fatalf("默认值不符：%+v", cfg)
	}
	if got := cfg.AllowedHosts; len(got) != 2 || got[0] != "127.0.0.1:17860" || got[1] != "localhost:17860" {
		t.Fatalf("未按端口派生 Host 白名单：%v", got)
	}
	if !cfg.IsCacheEnabled() {
		t.Fatal("默认应启用缓存")
	}
	if strings.Contains(cfg.RedactedDatabaseURL(), "secret") {
		t.Fatalf("日志中的连接串未隐藏口令：%s", cfg.RedactedDatabaseURL())
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://novel:secret@localhost:5432/novel")
	t.Setenv("APP_PORT", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("非法端口应返回错误")
	}
}

func TestLoadRejectsMissingDatabaseURL(t *testing.T) {
	unsetEnv(t, "DATABASE_URL")

	if _, err := Load(); err == nil {
		t.Fatal("缺少 DATABASE_URL 应返回错误")
	}
}

func TestLoadTreatsEmptyRedisAsDisabled(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://novel:secret@localhost:5432/novel")
	t.Setenv("REDIS_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回错误：%v", err)
	}
	if cfg.IsCacheEnabled() {
		t.Fatal("空 REDIS_ADDR 应视为禁用缓存")
	}
}
