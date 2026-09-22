// Package cleaningrecord 清淤记录录入模块：记录每次实际清淤的作业数据。
package cleaningrecord

import (
	"time"

	"github.com/drainage/desilting/internal/shared/date"
)

// CleaningRecord 清淤记录。
//
// 主表始终保存记录的「最新版本」：所有履历、汇总与看板统计都直接聚合本表，
// 天然按最新版本计算。历史版本不可变地保存在 CleaningRecordRevision 中，
// 每次修改都会先把修改前的数值留痕，再与本表更新在同一个事务内提交。
type CleaningRecord struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Code               string    `gorm:"size:32;uniqueIndex;not null" json:"code"`
	TaskID             uint      `gorm:"index;not null" json:"taskId"`
	CleanedAt          date.Date `gorm:"type:date;index;not null" json:"cleanedAt"`
	LengthM            float64   `gorm:"not null" json:"lengthM"`
	SludgeVolumeM3     float64   `gorm:"not null" json:"sludgeVolumeM3"`
	WaterVolumeM3      float64   `json:"waterVolumeM3"`
	PersonnelCount     int       `gorm:"not null" json:"personnelCount"`
	Method             string    `gorm:"size:24" json:"method"`
	Equipment          string    `gorm:"size:128" json:"equipment"`
	Weather            string    `gorm:"size:16" json:"weather"`
	SludgeDisposalSite string    `gorm:"size:128" json:"sludgeDisposalSite"`
	SafetyMeasures     string    `gorm:"type:text" json:"safetyMeasures"`
	ProblemFound       string    `gorm:"type:text" json:"problemFound"`
	RecorderName       string    `gorm:"size:32" json:"recorderName"`
	Remark             string    `gorm:"type:text" json:"remark"`
	// Version 当前版本号，录入时为 1，每次修改 +1，用于乐观并发控制。
	Version   int       `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName 指定表名。
func (CleaningRecord) TableName() string {
	return "cleaning_records"
}

// 履历动作类型。
const (
	RevisionActionCreate = "create" // 首次录入
	RevisionActionUpdate = "update" // 修改
)

// VersionSnapshot 某一版本的业务字段快照。
//
// 履历表以 JSON 文本保存快照（不与主表字段一一建列），字段新增时旧版本仍可完整反序列化，
// 任意两个版本的差异都可以直接比对快照得出。
type VersionSnapshot struct {
	CleanedAt          date.Date `json:"cleanedAt"`
	LengthM            float64   `json:"lengthM"`
	SludgeVolumeM3     float64   `json:"sludgeVolumeM3"`
	WaterVolumeM3      float64   `json:"waterVolumeM3"`
	PersonnelCount     int       `json:"personnelCount"`
	Method             string    `json:"method"`
	Equipment          string    `json:"equipment"`
	Weather            string    `json:"weather"`
	SludgeDisposalSite string    `json:"sludgeDisposalSite"`
	SafetyMeasures     string    `json:"safetyMeasures"`
	ProblemFound       string    `json:"problemFound"`
	RecorderName       string    `json:"recorderName"`
	Remark             string    `json:"remark"`
}

// CleaningRecordRevision 清淤记录的版本履历。
//
// 一行代表记录的一个版本：录入时写入第 1 版（动作为 create），
// 此后每次修改在同一事务内先追加新版本行（动作为 update，记录修改人、时间、原因），
// 再更新主表。履历行只能写入不能修改、不能单独删除，保证「数值已变必有对应记录」。
type CleaningRecordRevision struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	RecordID uint   `gorm:"index:idx_record_version,unique;not null" json:"recordId"`
	Version  int    `gorm:"index:idx_record_version,unique;not null" json:"version"`
	Action   string `gorm:"size:16;not null" json:"action"`
	Snapshot string `gorm:"type:text;not null" json:"-"`
	// ChangedBy 产生该版本的操作人（录入人 / 修改人）。
	ChangedBy    string    `gorm:"size:32;not null" json:"changedBy"`
	ChangeReason string    `gorm:"size:255;not null;default:''" json:"changeReason"`
	ChangedAt    time.Time `gorm:"not null" json:"changedAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

// TableName 指定表名。
func (CleaningRecordRevision) TableName() string {
	return "cleaning_record_revisions"
}

// snapshotOf 取记录当前业务字段的快照。
func snapshotOf(record *CleaningRecord) VersionSnapshot {
	return VersionSnapshot{
		CleanedAt:          record.CleanedAt,
		LengthM:            record.LengthM,
		SludgeVolumeM3:     record.SludgeVolumeM3,
		WaterVolumeM3:      record.WaterVolumeM3,
		PersonnelCount:     record.PersonnelCount,
		Method:             record.Method,
		Equipment:          record.Equipment,
		Weather:            record.Weather,
		SludgeDisposalSite: record.SludgeDisposalSite,
		SafetyMeasures:     record.SafetyMeasures,
		ProblemFound:       record.ProblemFound,
		RecorderName:       record.RecorderName,
		Remark:             record.Remark,
	}
}
