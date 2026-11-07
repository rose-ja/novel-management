package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// Up 执行全部未应用迁移；迁移文件随二进制嵌入。
func Up(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(files)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("设置迁移方言失败：%w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("数据库迁移失败：%w", err)
	}
	return nil
}
