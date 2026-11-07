package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// AdvisoryLockID 与访问端口一致，保证同一数据目录只允许一个应用实例。
const AdvisoryLockID = 17860

// ErrLockBusy 表示另一个实例或恢复任务持有数据库锁。
var ErrLockBusy = errors.New("另一个实例正在运行或正在恢复")

type DB struct {
	Gorm *gorm.DB
	SQL  *sql.DB
}

// Open 连接 PostgreSQL，最多重试 retry 指定的时长。
func Open(ctx context.Context, dsn string, retry time.Duration, log *slog.Logger) (*DB, error) {
	deadline := time.Now().Add(retry)
	var lastErr error

	for {
		db, err := tryOpen(ctx, dsn)
		if err == nil {
			return db, nil
		}
		lastErr = err

		if time.Now().After(deadline) {
			break
		}
		log.Warn("数据库尚未就绪，1 秒后重试", "error", err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return nil, fmt.Errorf("连接数据库超时（%s）：%w", retry, lastErr)
}

func tryOpen(ctx context.Context, dsn string) (*DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &DB{Gorm: gdb, SQL: sqlDB}, nil
}

// TryAdvisoryLock 获取会话级 advisory lock；成功后返回释放函数。
func (d *DB) TryAdvisoryLock(ctx context.Context, id int64) (func(), error) {
	conn, err := d.SQL.Conn(ctx)
	if err != nil {
		return nil, err
	}

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", id).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !acquired {
		_ = conn.Close()
		return nil, ErrLockBusy
	}

	return func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", id)
		_ = conn.Close()
	}, nil
}
