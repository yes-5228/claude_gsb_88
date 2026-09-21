// Package cleaningtask 清淤任务登记模块：负责把管段台账上的清淤需求登记成可跟踪的任务。
package cleaningtask

import (
	"time"

	"github.com/drainage/desilting/internal/shared/date"
)

// 任务优先级。
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// 任务来源。
const (
	SourcePlan       = "plan"
	SourceInspection = "inspection"
	SourceComplaint  = "complaint"
	SourceFlood      = "flood"
)

// 清淤方式。
const (
	MethodHighPressure = "high_pressure"
	MethodWinch        = "winch"
	MethodGrab         = "grab"
	MethodManual       = "manual"
	MethodRobot        = "robot"
)

// CleaningTask 清淤任务。
type CleaningTask struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Code          string     `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Title         string     `gorm:"size:128;not null" json:"title"`
	PipeSegmentID uint       `gorm:"index;not null" json:"pipeSegmentId"`
	Priority      string     `gorm:"size:16;index;not null;default:normal" json:"priority"`
	Source        string     `gorm:"size:16;index;not null;default:plan" json:"source"`
	Method        string     `gorm:"size:24" json:"method"`
	PlanStartDate date.Date  `gorm:"type:date;index;not null" json:"planStartDate"`
	PlanEndDate   date.Date  `gorm:"type:date;index;not null" json:"planEndDate"`
	TeamName      string     `gorm:"size:64" json:"teamName"`
	LeaderName    string     `gorm:"size:32" json:"leaderName"`
	LeaderPhone   string     `gorm:"size:32" json:"leaderPhone"`
	Status        string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	Description   string     `gorm:"type:text" json:"description"`
	StartedAt     *time.Time `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
	AcceptedAt    *time.Time `json:"acceptedAt"`
	CancelReason  string     `gorm:"size:255" json:"cancelReason"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// TableName 指定表名。
func (CleaningTask) TableName() string {
	return "cleaning_tasks"
}
