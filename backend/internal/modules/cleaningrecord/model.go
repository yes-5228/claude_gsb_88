// Package cleaningrecord 清淤记录录入模块：记录每次实际清淤的作业数据。
package cleaningrecord

import (
	"time"

	"github.com/drainage/desilting/internal/shared/date"
)

// CleaningRecord 清淤记录。
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
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// TableName 指定表名。
func (CleaningRecord) TableName() string {
	return "cleaning_records"
}

// CleaningRecordRevision 清淤记录修改留痕。
//
// 每次调整记录时，把修改前的数值快照、修改人、修改时间与修改原因保存为一条留痕；
// 记录表里的当前行永远是最新版本，因此所有统计天然按最新版本计算。
// 版本号从 1 开始（首次录入的值即 v1），当前版本号 = 留痕条数 + 1。
type CleaningRecordRevision struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	RecordID           uint      `gorm:"uniqueIndex:uk_record_version;not null" json:"recordId"`
	Version            int       `gorm:"uniqueIndex:uk_record_version;not null" json:"version"`
	CleanedAt          date.Date `gorm:"type:date;not null" json:"cleanedAt"`
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
	EditorName         string    `gorm:"size:32;not null" json:"editorName"`
	ChangeReason       string    `gorm:"size:200;not null" json:"changeReason"`
	CreatedAt          time.Time `json:"createdAt"`
}

// TableName 指定表名。
func (CleaningRecordRevision) TableName() string {
	return "cleaning_record_revisions"
}
