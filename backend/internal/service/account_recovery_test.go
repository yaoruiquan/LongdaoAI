package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRecoveryService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	// 创建测试账号
	acc1, err := client.Account.Create().
		SetName("Test Account 1").
		SetPlatform("openai").
		SetType("apikey").
		SetCredentials(map[string]interface{}{"api_key": "sk-test123"}).
		SetStatus("error").
		SetSchedulable(false).
		SetTempUnschedulableUntil(time.Now().Add(-1 * time.Hour)). // 1小时前过期
		SetTempUnschedulableReason("Test cooldown").
		Save(ctx)
	require.NoError(t, err)

	acc2, err := client.Account.Create().
		SetName("Test Account 2").
		SetPlatform("openai").
		SetType("apikey").
		SetCredentials(map[string]interface{}{"api_key": "sk-test456"}).
		SetStatus("error").
		SetSchedulable(false).
		SetTempUnschedulableUntil(time.Now().Add(1 * time.Hour)). // 1小时后过期（还未到期）
		SetTempUnschedulableReason("Test cooldown").
		Save(ctx)
	require.NoError(t, err)

	acc3, err := client.Account.Create().
		SetName("Test Account 3").
		SetPlatform("openai").
		SetType("apikey").
		SetCredentials(map[string]interface{}{"api_key": "sk-test789"}).
		SetStatus("active").
		SetSchedulable(true).
		Save(ctx)
	require.NoError(t, err)

	// 创建服务并执行恢复
	svc := NewAccountRecoveryService(client, nil)
	svc.recoverAccounts()

	// 验证结果
	recovered1, err := client.Account.Get(ctx, acc1.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", recovered1.Status, "Account 1 should be recovered")
	assert.True(t, recovered1.Schedulable, "Account 1 should be schedulable")
	assert.Nil(t, recovered1.TempUnschedulableUntil, "Account 1 temp_unschedulable_until should be cleared")

	notRecovered, err := client.Account.Get(ctx, acc2.ID)
	require.NoError(t, err)
	assert.Equal(t, "error", notRecovered.Status, "Account 2 should NOT be recovered (not expired yet)")
	assert.False(t, notRecovered.Schedulable, "Account 2 should NOT be schedulable")

	unchanged, err := client.Account.Get(ctx, acc3.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", unchanged.Status, "Account 3 should remain active")
	assert.True(t, unchanged.Schedulable, "Account 3 should remain schedulable")
}
