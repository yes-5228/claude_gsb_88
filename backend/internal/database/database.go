// Package database 负责数据库连接、表结构迁移。
package database

import (
	"context"
	"encoding/json"
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
	"github.com/drainage/desilting/internal/shared/date"
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
	if err := db.AutoMigrate(
		&pipesegment.PipeSegment{},
		&cleaningtask.CleaningTask{},
		&cleaningrecord.CleaningRecord{},
		&cleaningrecord.CleaningRecordRevision{},
		&acceptance.AcceptanceRecord{},
	); err != nil {
		return err
	}
	return BackfillRecordRevisions(db)
}

// BackfillRecordRevisions 为版本履历功能上线前已存在、且没有履历的记录补写第 1 版。
//
// 幂等：只处理 cleaning_record_revisions 中没有任何版本行的记录，
// 重复执行不会产生重复履历。用于保证「每条记录至少有一个版本可对比」。
func BackfillRecordRevisions(db *gorm.DB) error {
	type legacyRow struct {
		ID                 uint
		TaskID             uint
		CleanedAt          date.Date
		LengthM            float64
		SludgeVolumeM3     float64
		WaterVolumeM3      float64
		PersonnelCount     int
		Method             string
		Equipment          string
		Weather            string
		SludgeDisposalSite string
		SafetyMeasures     string
		ProblemFound       string
		RecorderName       string
		Remark             string
		Version            int
		CreatedAt          time.Time
	}
	rows := make([]legacyRow, 0)
	err := db.Raw(`SELECT r.* FROM cleaning_records AS r
		LEFT JOIN cleaning_record_revisions AS v ON v.record_id = r.id
		WHERE v.id IS NULL`).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("查询待回填履历的清淤记录失败: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			row := rows[i]
			snapshot := cleaningrecord.VersionSnapshot{
				CleanedAt:          row.CleanedAt,
				LengthM:            row.LengthM,
				SludgeVolumeM3:     row.SludgeVolumeM3,
				WaterVolumeM3:      row.WaterVolumeM3,
				PersonnelCount:     row.PersonnelCount,
				Method:             row.Method,
				Equipment:          row.Equipment,
				Weather:            row.Weather,
				SludgeDisposalSite: row.SludgeDisposalSite,
				SafetyMeasures:     row.SafetyMeasures,
				ProblemFound:       row.ProblemFound,
				RecorderName:       row.RecorderName,
				Remark:             row.Remark,
			}
			raw, err := json.Marshal(snapshot)
			if err != nil {
				return fmt.Errorf("序列化历史版本快照失败: %w", err)
			}
			changedAt := row.CreatedAt
			if changedAt.IsZero() {
				changedAt = time.Now()
			}
			version := row.Version
			if version <= 0 {
				version = 1
			}
			revision := cleaningrecord.CleaningRecordRevision{
				RecordID:     row.ID,
				Version:      version,
				Action:       cleaningrecord.RevisionActionCreate,
				Snapshot:     string(raw),
				ChangedBy:    fallbackName(row.RecorderName),
				ChangeReason: "历史数据补录",
				ChangedAt:    changedAt,
			}
			if err := tx.WithContext(context.Background()).Create(&revision).Error; err != nil {
				return fmt.Errorf("补录清淤记录版本履历失败: %w", err)
			}
			if row.Version <= 0 {
				if err := tx.Model(&cleaningrecord.CleaningRecord{}).
					Where("id = ? AND (version = 0 OR version IS NULL)", row.ID).
					Update("version", 1).Error; err != nil {
					return fmt.Errorf("回填记录版本号失败: %w", err)
				}
			}
		}
		return nil
	})
}

func fallbackName(name string) string {
	if name == "" {
		return "系统"
	}
	return name
}
