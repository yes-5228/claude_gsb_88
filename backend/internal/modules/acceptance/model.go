// Package acceptance 验收记录模块：对已完工的清淤任务做质量验收，并驱动任务闭环。
package acceptance

import (
	"time"

	"github.com/drainage/desilting/internal/shared/date"
)

// 验收结论。
const (
	ResultPass   = "pass"   // 合格
	ResultRework = "rework" // 需整改
)

// AcceptanceRecord 验收记录。验收结论一经登记不可修改，只能补充整改情况，
// 以保证验收过程的严肃性与可追溯性。
type AcceptanceRecord struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Code             string     `gorm:"size:32;uniqueIndex;not null" json:"code"`
	TaskID           uint       `gorm:"index;not null" json:"taskId"`
	CleaningRecordID *uint      `gorm:"index" json:"cleaningRecordId"`
	AcceptedAt       date.Date  `gorm:"type:date;index;not null" json:"acceptedAt"`
	InspectorName    string     `gorm:"size:32;not null" json:"inspectorName"`
	InspectorOrg     string     `gorm:"size:128" json:"inspectorOrg"`
	Result           string     `gorm:"size:16;index;not null" json:"result"`
	Score            int        `gorm:"not null" json:"score"`
	ResidualSludgeMm float64    `json:"residualSludgeMm"`
	Issues           string     `gorm:"type:text" json:"issues"`
	Rectification    string     `gorm:"type:text" json:"rectification"`
	RectifyDeadline  *date.Date `gorm:"type:date" json:"rectifyDeadline"`
	RectifiedAt      *date.Date `gorm:"type:date" json:"rectifiedAt"`
	Remark           string     `gorm:"type:text" json:"remark"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// TableName 指定表名。
func (AcceptanceRecord) TableName() string {
	return "acceptance_records"
}
