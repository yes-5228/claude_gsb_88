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
