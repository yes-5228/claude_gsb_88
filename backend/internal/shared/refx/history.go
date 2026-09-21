package refx

import (
	"context"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/num"
)

// HistoryItem 管段的清淤履历：一次任务串起"计划 -> 清淤记录 -> 验收结论"。
type HistoryItem struct {
	TaskID           uint      `json:"taskId"`
	TaskCode         string    `json:"taskCode"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority"`
	TeamName         string    `json:"teamName"`
	PlanStartDate    date.Date `json:"planStartDate"`
	PlanEndDate      date.Date `json:"planEndDate"`
	RecordCount      int64     `json:"recordCount"`
	SludgeVolumeM3   float64   `json:"sludgeVolumeM3"`
	CleanedLengthM   float64   `json:"cleanedLengthM"`
	AcceptanceResult string    `json:"acceptanceResult"`
	AcceptedAt       date.Date `json:"acceptedAt"`
}

// HistoryForSegment 查询某管段下全部任务的清淤履历（按计划开始日期倒序）。
func HistoryForSegment(ctx context.Context, db *gorm.DB, segmentID uint) ([]HistoryItem, error) {
	items := make([]HistoryItem, 0)
	err := db.WithContext(ctx).Table(TableCleaningTasks+" AS t").
		Select(`t.id AS task_id, t.code AS task_code, t.title, t.status, t.priority, t.team_name,
			t.plan_start_date, t.plan_end_date,
			COALESCE(r.record_count, 0) AS record_count,
			COALESCE(r.sludge_volume, 0) AS sludge_volume_m3,
			COALESCE(r.cleaned_length, 0) AS cleaned_length_m,
			COALESCE(ac.result, '') AS acceptance_result,
			ac.accepted_at`).
		Joins(`LEFT JOIN (
			SELECT task_id, COUNT(*) AS record_count,
				SUM(sludge_volume_m3) AS sludge_volume,
				SUM(length_m) AS cleaned_length
			FROM `+TableCleaningRecords+` GROUP BY task_id
		) AS r ON r.task_id = t.id`).
		Joins(`LEFT JOIN (
			SELECT a.task_id, a.result, a.accepted_at
			FROM `+TableAcceptanceRecords+` AS a
			INNER JOIN (
				SELECT task_id, MAX(id) AS max_id FROM `+TableAcceptanceRecords+` GROUP BY task_id
			) AS latest ON latest.max_id = a.id
		) AS ac ON ac.task_id = t.id`).
		Where("t.pipe_segment_id = ?", segmentID).
		Order("t.plan_start_date DESC, t.id DESC").
		Scan(&items).Error
	for i := range items {
		items[i].SludgeVolumeM3 = num.Round2(items[i].SludgeVolumeM3)
		items[i].CleanedLengthM = num.Round2(items[i].CleanedLengthM)
	}
	return items, err
}

// SludgeTotals 全局清淤量统计（用于管段台账列表的展示与统计口径统一）。
type SludgeTotals struct {
	RecordCount    int64   `json:"recordCount"`
	SludgeVolumeM3 float64 `json:"sludgeVolumeM3"`
	CleanedLengthM float64 `json:"cleanedLengthM"`
}

// SludgeTotalsForSegment 统计某管段累计清淤量。
func SludgeTotalsForSegment(ctx context.Context, db *gorm.DB, segmentID uint) (SludgeTotals, error) {
	var totals SludgeTotals
	err := db.WithContext(ctx).Table(TableCleaningRecords+" AS r").
		Select(`COUNT(*) AS record_count,
			COALESCE(SUM(r.sludge_volume_m3), 0) AS sludge_volume_m3,
			COALESCE(SUM(r.length_m), 0) AS cleaned_length_m`).
		Joins("INNER JOIN "+TableCleaningTasks+" AS t ON t.id = r.task_id").
		Where("t.pipe_segment_id = ?", segmentID).
		Scan(&totals).Error
	return totals, err
}
