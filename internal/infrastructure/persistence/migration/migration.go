package migration

import (
	"context"

	"github.com/go-gormigrate/gormigrate/v2"

	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/transactions"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
)

// 定义数据库迁移
var migrations []*gormigrate.Migration

func margeMigrations() {

}

// Migrate 执行数据库迁移
func Migrate(ctx context.Context, tx transactions.TransactionManager) error {
	logger.Info(ctx, "开始数据库迁移")

	margeMigrations()
	if len(migrations) == 0 {
		return nil
	}

	// 自动迁移数据库表结构
	err := gormigrate.New(tx.GetTx(ctx), gormigrate.DefaultOptions, migrations).Migrate()
	if err != nil {
		logger.Error(ctx, "数据库迁移失败", logger.ErrorField(err))
		return err
	}

	logger.Info(ctx, "数据库迁移完成")
	return nil
}

// Rollback 执行数据库回滚
func Rollback(ctx context.Context, tx transactions.TransactionManager, version ...string) error {
	logger.Info(ctx, "开始数据库迁移回滚")

	margeMigrations()
	if len(migrations) == 0 {
		return nil
	}

	var err error
	if len(version) > 0 {
		err = gormigrate.New(tx.GetTx(ctx), gormigrate.DefaultOptions, migrations).RollbackTo(version[0])
	} else {
		err = gormigrate.New(tx.GetTx(ctx), gormigrate.DefaultOptions, migrations).RollbackLast()
	}
	if err != nil {
		logger.Error(ctx, "数据库迁移回滚失败", logger.ErrorField(err))
		return err
	}

	logger.Info(ctx, "数据库迁移回滚完成")
	return nil
}

// Seed 填充初始数据
func Seed(ctx context.Context, tx transactions.TransactionManager) error {
	logger.Info(ctx, "开始填充初始数据")
	logger.Info(ctx, "初始数据填充完成")
	return nil
}
