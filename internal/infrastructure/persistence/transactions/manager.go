package transactions

import (
	"context"

	"gorm.io/gorm"
)

// Manager 数据库事务管理器
type Manager struct {
	db *gorm.DB
}

func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

// WithTransaction 在事务中执行 fn，自动提交或回滚
func (m *Manager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// DB 从 ctx 取事务 DB，否则返回普通 DB
func (m *Manager) DB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return m.db.WithContext(ctx)
}

type txKey struct{}
