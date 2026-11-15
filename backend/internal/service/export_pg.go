package service

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"novel-management/backend/internal/model"
)

// PGExportDumper 使用 pg_dump/pg_restore 产出真实转储；不含回收站时借助临时库过滤。
type PGExportDumper struct {
	db     *gorm.DB
	dumper Dumper
}

func NewPGExportDumper(db *gorm.DB) *PGExportDumper {
	return &PGExportDumper{db: db, dumper: PGDumper{}}
}

func (p *PGExportDumper) Dump(ctx context.Context, dsn, outPath string) error {
	return p.dumper.Dump(ctx, dsn, outPath)
}

// DumpWithoutTrash 复制当前库到临时库，物理删除软删除根与隐藏关系后再转储。
func (p *PGExportDumper) DumpWithoutTrash(ctx context.Context, dsn, outPath string) error {
	name := "novel_export_tmp_" + strings.ToLower(model.NewID())
	if len(name) > 63 {
		name = name[:63]
	}
	if err := p.db.WithContext(ctx).Exec(`CREATE DATABASE "` + name + `"`).Error; err != nil {
		return fmt.Errorf("创建临时数据库失败：%w", err)
	}
	defer func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := p.db.WithContext(dropCtx).Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`).Error; err != nil {
			// 临时库删除失败不影响导出结果，由运维按日志清理。
			fmt.Fprintf(os.Stderr, "清理临时数据库 %s 失败：%v\n", name, err)
		}
	}()

	tempDSN, err := replaceDatabaseName(dsn, name)
	if err != nil {
		return err
	}

	sourceDump := outPath + ".source"
	defer func() { _ = os.Remove(sourceDump) }()
	if err := p.dumper.Dump(ctx, dsn, sourceDump); err != nil {
		return err
	}
	if err := runPGCommand(ctx, tempDSN, "pg_restore",
		"--no-owner", "--no-privileges", sourceDump); err != nil {
		return err
	}

	tempDB, err := gorm.Open(postgres.Open(tempDSN), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		return fmt.Errorf("连接临时数据库失败：%w", err)
	}
	sqlDB, err := tempDB.DB()
	if err != nil {
		return err
	}
	if err := cleanupTrashData(ctx, tempDB); err != nil {
		_ = sqlDB.Close()
		return err
	}
	if err := sqlDB.Close(); err != nil {
		return err
	}

	return p.dumper.Dump(ctx, tempDSN, outPath)
}

// cleanupTrashData 按「关系行 → 软删除子表 → 软删除根表」顺序清理，避免外键阻断。
func cleanupTrashData(ctx context.Context, db *gorm.DB) error {
	statements := []string{
		`DELETE FROM ranking_seats
		 WHERE character_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)
		    OR ranking_id IN (SELECT id FROM rankings WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM character_locations
		 WHERE character_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)
		    OR location_id IN (SELECT id FROM locations WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM location_factions
		 WHERE location_id IN (SELECT id FROM locations WHERE deleted_at IS NOT NULL)
		    OR faction_id IN (SELECT id FROM factions WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM character_system_refs
		 WHERE character_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)
		    OR node_id IN (SELECT id FROM system_nodes WHERE deleted_at IS NOT NULL)
		    OR category_id IN (SELECT id FROM system_categories WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM character_names
		 WHERE character_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM chronicles
		 WHERE owner_type = 'character'
		   AND owner_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM media_assets
		 WHERE entity_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)`,
		`UPDATE factions SET leader_character_id = NULL
		 WHERE leader_character_id IN (SELECT id FROM characters WHERE deleted_at IS NOT NULL)`,
		`DELETE FROM chronicles WHERE deleted_at IS NOT NULL`,
		`DELETE FROM media_assets WHERE deleted_at IS NOT NULL`,
		`DELETE FROM rankings WHERE deleted_at IS NOT NULL`,
		`DELETE FROM characters WHERE deleted_at IS NOT NULL`,
		`DELETE FROM factions WHERE deleted_at IS NOT NULL`,
		`DELETE FROM locations WHERE deleted_at IS NOT NULL`,
		`DELETE FROM system_nodes WHERE deleted_at IS NOT NULL`,
		`DELETE FROM system_categories WHERE deleted_at IS NOT NULL`,
		`DELETE FROM works WHERE deleted_at IS NOT NULL`,
		`DELETE FROM undo_tokens`,
		`DELETE FROM search_documents`,
	}
	for _, statement := range statements {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("清理回收站数据失败：%w", err)
		}
	}
	return nil
}

func replaceDatabaseName(dsn, name string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("解析 DATABASE_URL 失败：%w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", fmt.Errorf("导出仅支持 postgres 连接串，当前为 %q", parsed.Scheme)
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

// runPGCommand 通过环境变量传入连接参数，避免口令出现在进程列表。
func runPGCommand(ctx context.Context, dsn, name string, args ...string) error {
	env, database, err := pgEnv(dsn)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, name, append(args, "-d", database)...)
	command.Env = env

	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s 失败：%w（%s）", name, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func pgEnv(dsn string) ([]string, string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, "", fmt.Errorf("解析 DATABASE_URL 失败：%w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return nil, "", fmt.Errorf("仅支持 postgres 连接串，当前为 %q", parsed.Scheme)
	}
	database := strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		return nil, "", fmt.Errorf("DATABASE_URL 缺少数据库名")
	}

	env := os.Environ()
	if host := parsed.Hostname(); host != "" {
		env = append(env, "PGHOST="+host)
	}
	if port := parsed.Port(); port != "" {
		env = append(env, "PGPORT="+port)
	}
	if parsed.User != nil {
		if user := parsed.User.Username(); user != "" {
			env = append(env, "PGUSER="+user)
		}
		if password, ok := parsed.User.Password(); ok {
			env = append(env, "PGPASSWORD="+password)
		}
	}
	if sslMode := parsed.Query().Get("sslmode"); sslMode != "" {
		env = append(env, "PGSSLMODE="+sslMode)
	}
	return env, database, nil
}
