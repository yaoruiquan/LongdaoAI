package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // 注册 postgres driver，保证 sql.Open("postgres", ...) 可用

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
)

// migrateTimeout 是独立迁移命令的整体超时时间。
// 迁移涉及 advisory lock 等待与多条 SQL 执行，给足 5 分钟余量。
const migrateTimeout = 5 * time.Minute

// runMigrate 是 `-migrate` 命令的入口。
//
// 它不依赖 .installed 标记，可在每次发布时独立触发，用于补执行历史遗漏的迁移：
//  1. 加载启动阶段配置
//  2. 用配置里的 DSN 建立数据库连接并探活
//  3. 调用 repository.ApplyMigrations 应用所有待执行迁移
//
// 返回 error 时由调用方（main）负责以非零退出码结束进程，
// 以便发布脚本据此阻止启动新版本应用。
func runMigrate() error {
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	return migrateWithDSN(ctx, cfg.Database.DSN(), repository.ApplyMigrations)
}

// migrateWithDSN 用给定 DSN 建立连接后执行迁移。
//
// 该函数从 runMigrate 中拆出，便于在无真实数据库的环境下对
// 「连接失败」等分支编写单元测试（apply 以参数注入，方便替换）。
func migrateWithDSN(ctx context.Context, dsn string, apply func(context.Context, *sql.DB) error) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	return migrateDB(ctx, db, apply)
}

// migrateDB 在已有连接上执行探活与迁移。
//
// 独立成函数是为了让 apply 分支可被单测覆盖（可配合 sqlmock 使用）。
func migrateDB(ctx context.Context, db *sql.DB, apply func(context.Context, *sql.DB) error) error {
	// 迁移前先探活，确保数据库可达，避免锁等待阶段才暴露连接问题。
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if err := apply(ctx, db); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
