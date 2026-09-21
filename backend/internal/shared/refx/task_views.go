package refx

import (
	"context"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/num"
)

// RecordTotals 某个任务的清淤记录汇总。
type RecordTotals struct {
	RecordCount     int64     `json:"recordCount"`
	SludgeVolumeM3  float64   `json:"sludgeVolumeM3"`
	CleanedLengthM  float64   `json:"cleanedLengthM"`
	LatestCleanedAt date.Date `json:"latestCleanedAt"`
}

// AcceptanceBrief 某个任务的最新验收结论。
type AcceptanceBrief struct {
	ID              uint      `json:"id"`
	Code            string    `json:"code"`
	Result          string    `json:"result"`
	AcceptedAt      date.Date `json:"acceptedAt"`
	InspectorName   string    `json:"inspectorName"`
	InspectorOrg    string    `json:"inspectorOrg"`
	Score           int       `json:"score"`
	Issues          string    `json:"issues"`
	RectifyDeadline date.Date `json:"rectifyDeadline"`
	RectifiedAt     date.Date `json:"rectifiedAt"`
}

// TotalsByTaskID 汇总单个任务的清淤记录。
func TotalsByTaskID(ctx context.Context, db *gorm.DB, taskID uint) (RecordTotals, error) {
	totals := RecordTotals{}
	err := db.WithContext(ctx).Table(TableCleaningRecords).
		Select(`COUNT(*) AS record_count,
			COALESCE(SUM(sludge_volume_m3), 0) AS sludge_volume_m3,
			COALESCE(SUM(length_m), 0) AS cleaned_length_m,
			MAX(cleaned_at) AS latest_cleaned_at`).
		Where("task_id = ?", taskID).
		Scan(&totals).Error
	totals.SludgeVolumeM3 = num.Round2(totals.SludgeVolumeM3)
	totals.CleanedLengthM = num.Round2(totals.CleanedLengthM)
	return totals, err
}

// TotalsByTaskIDs 批量汇总多个任务的清淤记录，避免列表接口 N+1 查询。
func TotalsByTaskIDs(ctx context.Context, db *gorm.DB, taskIDs []uint) (map[uint]RecordTotals, error) {
	result := make(map[uint]RecordTotals, len(taskIDs))
	if len(taskIDs) == 0 {
		return result, nil
	}
	type row struct {
		TaskID          uint
		RecordCount     int64
		SludgeVolumeM3  float64
		CleanedLengthM  float64
		LatestCleanedAt *date.Date
	}
	rows := make([]row, 0, len(taskIDs))
	err := db.WithContext(ctx).Table(TableCleaningRecords).
		Select(`task_id, COUNT(*) AS record_count,
			COALESCE(SUM(sludge_volume_m3), 0) AS sludge_volume_m3,
			COALESCE(SUM(length_m), 0) AS cleaned_length_m,
			MAX(cleaned_at) AS latest_cleaned_at`).
		Where("task_id IN ?", taskIDs).
		Group("task_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		totals := RecordTotals{
			RecordCount:    item.RecordCount,
			SludgeVolumeM3: num.Round2(item.SludgeVolumeM3),
			CleanedLengthM: num.Round2(item.CleanedLengthM),
		}
		if item.LatestCleanedAt != nil {
			totals.LatestCleanedAt = *item.LatestCleanedAt
		}
		result[item.TaskID] = totals
	}
	return result, nil
}

// LatestAcceptanceForTask 查询任务最近一次验收记录。
func LatestAcceptanceForTask(ctx context.Context, db *gorm.DB, taskID uint) (*AcceptanceBrief, error) {
	var brief AcceptanceBrief
	err := db.WithContext(ctx).Table(TableAcceptanceRecords).
		Select(`id, code, result, accepted_at, inspector_name, inspector_org,
			score, issues, rectify_deadline, rectified_at`).
		Where("task_id = ?", taskID).
		Order("id DESC").
		Limit(1).
		Scan(&brief).Error
	if err != nil {
		return nil, err
	}
	if brief.ID == 0 {
		return nil, nil
	}
	return &brief, nil
}
