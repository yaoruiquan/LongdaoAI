package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// TestMigrateDB_Success 验证探活成功且 apply 成功时返回 nil。
func TestMigrateDB_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPing()

	applyCalled := false
	apply := func(ctx context.Context, gotDB *sql.DB) error {
		applyCalled = true
		require.Equal(t, db, gotDB)
		return nil
	}

	err = migrateDB(context.Background(), db, apply)
	require.NoError(t, err)
	require.True(t, applyCalled, "apply 应被调用")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigrateDB_PingFailure 验证探活失败时返回错误且不调用 apply。
func TestMigrateDB_PingFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPing().WillReturnError(errors.New("connection refused"))

	applyCalled := false
	apply := func(ctx context.Context, gotDB *sql.DB) error {
		applyCalled = true
		return nil
	}

	err = migrateDB(context.Background(), db, apply)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ping database")
	require.False(t, applyCalled, "探活失败时不应调用 apply")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigrateDB_ApplyFailure 验证探活成功但 apply 失败时错误被包装返回。
func TestMigrateDB_ApplyFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPing()

	apply := func(ctx context.Context, gotDB *sql.DB) error {
		return errors.New("checksum mismatch")
	}

	err = migrateDB(context.Background(), db, apply)
	require.Error(t, err)
	require.Contains(t, err.Error(), "apply migrations")
	require.Contains(t, err.Error(), "checksum mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigrateWithDSN_ConnectFailure 验证连接不可达时返回错误（无需真实数据库）。
// 使用已取消的 context 让探活立即失败，避免真实连接等待。
func TestMigrateWithDSN_ConnectFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	applyCalled := false
	apply := func(ctx context.Context, db *sql.DB) error {
		applyCalled = true
		return nil
	}

	// 指向一个不可达地址；配合已取消的 ctx，PingContext 应立即返回错误。
	dsn := "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable"
	err := migrateWithDSN(ctx, dsn, apply)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ping database")
	require.False(t, applyCalled, "连接失败时不应调用 apply")
}
