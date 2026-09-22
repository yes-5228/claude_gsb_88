package database_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/drainage/desilting/internal/database"
	"github.com/drainage/desilting/internal/modules/cleaningrecord"
	"github.com/drainage/desilting/internal/shared/date"
)

func newBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// TestBackfillRevisionsForLegacyRecords 模拟履历功能上线前的存量库：
// 已存在清淤记录但没有任何版本履历，执行迁移后应补写第 1 版，且重复执行幂等。
func TestBackfillRevisionsForLegacyRecords(t *testing.T) {
	db := newBackfillDB(t)

	// 先建出含履历表的完整结构（模拟应用升级后的 AutoMigrate）。
	if err := database.Migrate(db); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}

	// 手工插入一条「没有履历」的存量记录（version 给 0，模拟旧数据）。
	legacy := cleaningrecord.CleaningRecord{
		Code: "QJ20260101-0001", TaskID: 1, CleanedAt: date.MustParse("2026-01-01"),
		LengthM: 100, SludgeVolumeM3: 12.5, PersonnelCount: 4,
		Method: "manual", Weather: "sunny", RecorderName: "存量记录人",
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("插入存量记录失败: %v", err)
	}
	// 删除可能由其他逻辑写入的履历，确保起点是「无履历」。
	if err := db.Where("record_id = ?", legacy.ID).Delete(&cleaningrecord.CleaningRecordRevision{}).Error; err != nil {
		t.Fatalf("清理履历失败: %v", err)
	}

	if err := database.BackfillRecordRevisions(db); err != nil {
		t.Fatalf("回填履历失败: %v", err)
	}
	var count int64
	if err := db.Model(&cleaningrecord.CleaningRecordRevision{}).
		Where("record_id = ?", legacy.ID).Count(&count).Error; err != nil {
		t.Fatalf("查询履历失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("存量记录应补写 1 条履历，实际 %d", count)
	}

	var rev cleaningrecord.CleaningRecordRevision
	if err := db.Where("record_id = ?", legacy.ID).First(&rev).Error; err != nil {
		t.Fatalf("查询补录履历失败: %v", err)
	}
	if rev.Version != 1 || rev.Action != cleaningrecord.RevisionActionCreate {
		t.Fatalf("补录履历版本/动作不正确: %+v", rev)
	}
	if rev.ChangedBy != "存量记录人" {
		t.Fatalf("补录履历修改人应取记录人，实际 %q", rev.ChangedBy)
	}

	// 主表版本号应被回填为 1。
	var version int
	if err := db.Table("cleaning_records").Where("id = ?", legacy.ID).Select("version").Scan(&version).Error; err != nil {
		t.Fatalf("查询版本号失败: %v", err)
	}
	if version != 1 {
		t.Fatalf("存量记录版本号应回填为 1，实际 %d", version)
	}

	// 再次回填不应产生重复履历。
	if err := database.BackfillRecordRevisions(db); err != nil {
		t.Fatalf("二次回填失败: %v", err)
	}
	if err := db.Model(&cleaningrecord.CleaningRecordRevision{}).
		Where("record_id = ?", legacy.ID).Count(&count).Error; err != nil {
		t.Fatalf("二次查询履历失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("回填应幂等，期望仍为 1 条，实际 %d", count)
	}
}
