package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/account"
)

// AccountRecoveryService 定期恢复已过期的临时禁用账号
type AccountRecoveryService struct {
	client   *ent.Client
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewAccountRecoveryService 创建账号自动恢复服务
// interval: 检查间隔，默认 3 小时
func NewAccountRecoveryService(client *ent.Client, logger *slog.Logger) *AccountRecoveryService {
	return &AccountRecoveryService{
		client:   client,
		logger:   logger.With("component", "account_recovery"),
		interval: 3 * time.Hour, // 默认 3 小时检查一次
		stopCh:   make(chan struct{}),
	}
}

// Start 启动定时任务
func (s *AccountRecoveryService) Start() {
	s.wg.Add(1)
	go s.run()
	s.logger.Info("account recovery service started", "interval", s.interval)
}

// Stop 停止定时任务
func (s *AccountRecoveryService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	s.logger.Info("account recovery service stopped")
}

func (s *AccountRecoveryService) run() {
	defer s.wg.Done()

	// 启动时立即执行一次
	s.recoverAccounts()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.recoverAccounts()
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountRecoveryService) recoverAccounts() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()

	// 查询需要恢复的账号：
	// 1. deleted_at IS NULL（未删除）
	// 2. schedulable = false（当前不可调度）
	// 3. temp_unschedulable_until IS NOT NULL（有设置临时禁用时间）
	// 4. temp_unschedulable_until < NOW()（禁用时间已过期）
	affectedAccounts, err := s.client.Account.
		Update().
		Where(
			account.DeletedAtIsNil(),
			account.SchedulableEQ(false),
			account.TempUnschedulableUntilNotNil(),
			account.TempUnschedulableUntilLT(now),
		).
		SetStatus("active").
		SetSchedulable(true).
		ClearErrorMessage().
		ClearTempUnschedulableUntil().
		ClearTempUnschedulableReason().
		Save(ctx)

	if err != nil {
		s.logger.Error("failed to recover accounts",
			"error", err,
		)
		return
	}

	if affectedAccounts > 0 {
		s.logger.Info("recovered expired accounts",
			"count", affectedAccounts,
			"check_time", now,
		)
	} else {
		s.logger.Debug("no accounts need recovery",
			"check_time", now,
		)
	}
}
