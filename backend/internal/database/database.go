// Package database 负责数据库连接、表结构迁移。
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/drainage/desilting/internal/config"
	"github.com/drainage/desilting/internal/modules/acceptance"
	"github.com/drainage/desilting/internal/modules/cleaningrecord"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/modules/pipesegment"
)

// Open 根据配置建立数据库连接。
//
// 生产与 docker compose 使用 PostgreSQL；本地开发或单元测试可以切到 SQLite，
// 两种驱动共用同一套 model 与查询代码。
func Open(cfg *config.Config) (*gorm.DB, error) {
	dialector, err := dialectorFor(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:         gormlogger.Default.LogMode(gormlogger.Warn),
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().In(time.Local) },
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func dialectorFor(cfg *config.Config) (gorm.Dialector, error) {
	switch cfg.DBDriver {
	case config.DriverSQLite:
		if dir := filepath.Dir(cfg.DBPath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("创建 SQLite 数据目录失败: %w", err)
			}
		}
		return sqlite.Open(cfg.DBPath), nil
	case config.DriverPostgres:
		return postgres.Open(cfg.PostgresDSN()), nil
	default:
		return nil, fmt.Errorf("不支持的数据库驱动 %q，可选值为 %s / %s",
			cfg.DBDriver, config.DriverPostgres, config.DriverSQLite)
	}
}

// Migrate 建立/更新所有业务表。
//
// 表之间的引用关系由应用层在 service 中校验，因此这里不建外键约束，
// 便于后续按模块拆库时平滑迁移。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&pipesegment.PipeSegment{},
		&cleaningtask.CleaningTask{},
		&cleaningrecord.CleaningRecord{},
		&acceptance.AcceptanceRecord{},
	)
}
