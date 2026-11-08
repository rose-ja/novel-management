package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

// Dumper 抽象数据库导出，便于在单元测试中注入假实现。
type Dumper interface {
	Dump(ctx context.Context, dsn, outPath string) error
}

// PGDumper 调用运行时镜像内的 pg_dump（PostgreSQL 16 客户端）。
type PGDumper struct{}

func (PGDumper) Dump(ctx context.Context, dsn, outPath string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("解析 DATABASE_URL 失败：%w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("快照仅支持 postgres 连接串，当前为 %q", u.Scheme)
	}

	port := u.Port()
	if port == "" {
		port = "5432"
	}
	args := []string{
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"--file", outPath,
		"--host", u.Hostname(),
		"--port", port,
	}
	var password string
	if u.User != nil {
		if name := u.User.Username(); name != "" {
			args = append(args, "--username", name)
		}
		password, _ = u.User.Password()
	}
	if dbName := strings.TrimPrefix(u.Path, "/"); dbName != "" {
		args = append(args, dbName)
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	env := os.Environ()
	if password != "" {
		env = append(env, "PGPASSWORD="+password)
	}
	if sslMode := u.Query().Get("sslmode"); sslMode != "" {
		env = append(env, "PGSSLMODE="+sslMode)
	}
	cmd.Env = env

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump 失败：%w（%s）", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Snapshot 负责快照五件套的产出、记录与保留策略。
type Snapshot struct {
	db     *gorm.DB
	cfg    config.Config
	log    *slog.Logger
	dumper Dumper
	now    func() time.Time
}

func NewSnapshot(db *gorm.DB, cfg config.Config, log *slog.Logger, dumper Dumper) *Snapshot {
	return &Snapshot{db: db, cfg: cfg, log: log, dumper: dumper, now: time.Now}
}

type snapshotMeta struct {
	ID           string           `json:"id"`
	Reason       string           `json:"reason"`
	CreatedAt    string           `json:"created_at"`
	EntityCounts map[string]int64 `json:"entity_counts"`
}

type manifestFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type manifest struct {
	CreatedAt string         `json:"created_at"`
	Files     []manifestFile `json:"files"`
}

// Create 产出快照并落记录；任何失败都返回 ErrSnapshotFailed，调用方据此阻止破坏性操作。
func (s *Snapshot) Create(ctx context.Context, reason string) (*model.Snapshot, error) {
	created := s.now().UTC()
	id := model.NewID()
	dir := s.snapshotDir(created, reason, id)

	record := &model.Snapshot{
		ID:        id,
		Reason:    reason,
		Path:      dir,
		Status:    model.SnapshotStatusFailed,
		CreatedAt: created,
	}

	if err := s.produce(ctx, dir, record); err != nil {
		s.log.Error("快照失败", "reason", reason, "path", dir, "error", err)
		if insertErr := repository.NewSnapshots(s.db).Insert(ctx, record); insertErr != nil {
			s.log.Error("写入失败快照记录失败", "error", insertErr)
		}
		return nil, fmt.Errorf("%w: %v", model.ErrSnapshotFailed, err)
	}

	record.Status = model.SnapshotStatusOK
	if err := repository.NewSnapshots(s.db).Insert(ctx, record); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("%w: %v", model.ErrSnapshotFailed, err)
	}

	s.prune(ctx)
	return record, nil
}

// List 返回最近快照记录，最多 limit 条。
func (s *Snapshot) List(ctx context.Context, limit int) ([]model.Snapshot, error) {
	return repository.NewSnapshots(s.db).ListRecent(ctx, limit)
}

// snapshotDir 使用 {ts}-{reason}；同一秒内重复生成时追加随机后缀避免目录覆盖。
// 后缀取 ULID 末 6 位（随机段），同毫秒生成也不会碰撞。
func (s *Snapshot) snapshotDir(created time.Time, reason, id string) string {
	base := created.Format("20060102T150405Z") + "-" + reason
	dir := filepath.Join(s.cfg.DataDir, "snapshots", base)
	if _, err := os.Stat(dir); err == nil {
		suffix := id
		if len(suffix) > 6 {
			suffix = suffix[len(suffix)-6:]
		}
		dir = filepath.Join(s.cfg.DataDir, "snapshots", base+"-"+suffix)
	}
	return dir
}

// produce 写出五件套：app.dump、images/、snapshot.json、manifest.json、checksums.sha256。
func (s *Snapshot) produce(ctx context.Context, dir string, record *model.Snapshot) error {
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		return err
	}

	dumpPath := filepath.Join(dir, "app.dump")
	if err := s.dumper.Dump(ctx, s.cfg.DatabaseURL, dumpPath); err != nil {
		return err
	}
	info, err := os.Stat(dumpPath)
	if err != nil {
		return err
	}
	record.DBBytes = info.Size()

	imageCount, err := copyTree(filepath.Join(s.cfg.DataDir, "images"), filepath.Join(dir, "images"))
	if err != nil {
		return err
	}
	record.ImageCount = imageCount

	meta := snapshotMeta{
		ID:           record.ID,
		Reason:       record.Reason,
		CreatedAt:    record.CreatedAt.Format(time.RFC3339),
		EntityCounts: s.entityCounts(ctx),
	}
	if err := writeJSON(filepath.Join(dir, "snapshot.json"), meta); err != nil {
		return err
	}

	files, err := listFiles(dir)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "manifest.json"), manifest{
		CreatedAt: record.CreatedAt.Format(time.RFC3339),
		Files:     files,
	}); err != nil {
		return err
	}

	return writeChecksums(dir, filepath.Join(dir, "checksums.sha256"))
}

func (s *Snapshot) entityCounts(ctx context.Context) map[string]int64 {
	counts := map[string]int64{}
	if s.db == nil {
		return counts
	}
	for _, table := range []string{"works", "system_categories"} {
		var total int64
		if err := s.db.WithContext(ctx).Table(table).Count(&total).Error; err == nil {
			counts[table] = total
		}
	}
	return counts
}

// prune 只裁剪成功快照，保留 cfg.SnapshotKeep 份；失败记录留作审计。
func (s *Snapshot) prune(ctx context.Context) {
	keep := s.cfg.SnapshotKeep
	if keep < 1 {
		keep = model.SnapshotKeepDefault
	}

	items, err := repository.NewSnapshots(s.db).ListSuccessful(ctx)
	if err != nil {
		s.log.Warn("读取快照记录失败，跳过裁剪", "error", err)
		return
	}
	for index, item := range items {
		if index < keep {
			continue
		}
		if err := os.RemoveAll(item.Path); err != nil {
			s.log.Warn("删除过期快照目录失败", "path", item.Path, "error", err)
			continue
		}
		if err := repository.NewSnapshots(s.db).DeleteOne(ctx, item.ID); err != nil {
			s.log.Warn("删除过期快照记录失败", "id", item.ID, "error", err)
		}
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// copyTree 递归复制目录并返回文件数量；源目录不存在时视为 0。
func copyTree(src, dst string) (int, error) {
	entries, err := os.ReadDir(src)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		source := filepath.Join(src, entry.Name())
		target := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			nested, err := copyTree(source, target)
			if err != nil {
				return count, err
			}
			count += nested
			continue
		}
		if err := copyFile(source, target); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// listFiles 收集目录内全部文件的相对路径与大小（不含 checksums 自身）。
func listFiles(root string) ([]manifestFile, error) {
	var files []manifestFile
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "checksums.sha256" {
			return nil
		}
		files = append(files, manifestFile{Path: rel, Size: info.Size()})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, err
}

func writeChecksums(root, outPath string) error {
	files, err := listFiles(root)
	if err != nil {
		return err
	}
	var buf strings.Builder
	for _, file := range files {
		sum, err := fileSHA256(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil {
			return err
		}
		fmt.Fprintf(&buf, "%s  %s\n", sum, file.Path)
	}
	return os.WriteFile(outPath, []byte(buf.String()), 0o644)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
