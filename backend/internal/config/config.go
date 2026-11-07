package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPort  = 17860
	DefaultHost  = "0.0.0.0"
	Version      = "0.1.0"
	APIVersion   = "v1"
	CacheTTLMax  = 300
	DBRetryLimit = 60 * time.Second
)

// Config 是应用的全部运行时配置，来源为环境变量。
type Config struct {
	Host             string
	Port             int
	AllowedHosts     []string
	DataDir          string
	WebDir           string
	DatabaseURL      string
	RedisAddr        string
	RedisPassword    string
	CacheTTLSec      int
	MaxUploadMB      int
	SnapshotKeep     int
	ImportTTLMin     int
	ImportTimeoutMin int
	LogLevel         string
	TZ               string
	Version          string
}

// Addr 返回 HTTP 监听地址。
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// IsCacheEnabled 表示是否配置了 Redis；空地址视为禁用缓存。
func (c Config) IsCacheEnabled() bool {
	return strings.TrimSpace(c.RedisAddr) != ""
}

// RedactedDatabaseURL 用于日志与自检输出，隐藏口令。
func (c Config) RedactedDatabaseURL() string {
	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return "<invalid>"
	}
	if u.User != nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.UserPassword(u.User.Username(), "***")
		}
	}
	return u.String()
}

// Load 读取环境变量并校验；缺失关键配置时返回错误。
func Load() (Config, error) {
	cfg := Config{
		Host:          env("APP_HOST", DefaultHost),
		DataDir:       env("DATA_DIR", "/data"),
		WebDir:        env("WEB_DIR", "./web"),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisAddr:     env("REDIS_ADDR", "cache:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		LogLevel:      strings.ToUpper(env("LOG_LEVEL", "INFO")),
		TZ:            env("TZ", "Asia/Shanghai"),
		Version:       env("APP_VERSION", Version),
	}

	port, err := intEnv("APP_PORT", DefaultPort)
	if err != nil {
		return Config{}, err
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("APP_PORT 必须在 1-65535 之间，当前为 %d", port)
	}
	cfg.Port = port

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL 未设置")
	}
	if cfg.Host == "" {
		return Config{}, fmt.Errorf("APP_HOST 不能为空")
	}
	if cfg.DataDir == "" {
		return Config{}, fmt.Errorf("DATA_DIR 不能为空")
	}

	cfg.CacheTTLSec, err = intEnv("CACHE_TTL_SEC", 60)
	if err != nil {
		return Config{}, err
	}
	if cfg.CacheTTLSec < 1 || cfg.CacheTTLSec > CacheTTLMax {
		return Config{}, fmt.Errorf("CACHE_TTL_SEC 必须在 1-%d 之间，当前为 %d", CacheTTLMax, cfg.CacheTTLSec)
	}
	if cfg.MaxUploadMB, err = intEnv("MAX_UPLOAD_MB", 6144); err != nil {
		return Config{}, err
	}
	if cfg.SnapshotKeep, err = intEnv("SNAPSHOT_KEEP", 3); err != nil {
		return Config{}, err
	}
	if cfg.ImportTTLMin, err = intEnv("IMPORT_TTL_MIN", 120); err != nil {
		return Config{}, err
	}
	if cfg.ImportTimeoutMin, err = intEnv("IMPORT_TIMEOUT_MIN", 120); err != nil {
		return Config{}, err
	}

	hosts := strings.TrimSpace(env("ALLOWED_HOSTS", ""))
	if hosts == "" {
		cfg.AllowedHosts = []string{
			fmt.Sprintf("127.0.0.1:%d", port),
			fmt.Sprintf("localhost:%d", port),
		}
	} else {
		cfg.AllowedHosts = splitHosts(hosts)
		if len(cfg.AllowedHosts) == 0 {
			return Config{}, fmt.Errorf("ALLOWED_HOSTS 配置无效")
		}
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s 必须为整数，当前为 %q", key, raw)
	}
	return v, nil
}

func splitHosts(raw string) []string {
	parts := strings.Split(raw, ",")
	hosts := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			hosts = append(hosts, v)
		}
	}
	return hosts
}
