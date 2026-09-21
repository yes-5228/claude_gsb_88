package cleaningtask

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/shared/refx"
)

// ErrNotFound 任务不存在。
var ErrNotFound = errors.New("清淤任务不存在")

// ErrStateConflict 并发操作导致状态已变化。
var ErrStateConflict = errors.New("任务状态已变化，请刷新后重试")

// Repository 清淤任务数据访问。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// DB 暴露底层连接，供 service 做反向引用检查。
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// Create 新增任务。
func (r *Repository) Create(ctx context.Context, task *CleaningTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// Save 保存任务全部字段。
func (r *Repository) Save(ctx context.Context, task *CleaningTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete 物理删除任务。
func (r *Repository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&CleaningTask{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByID 按主键查询任务。
func (r *Repository) FindByID(ctx context.Context, id uint) (*CleaningTask, error) {
	var task CleaningTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// MaxCodeWithPrefix 查询某个前缀下已使用的最大任务编号，用于生成流水号。
func (r *Repository) MaxCodeWithPrefix(ctx context.Context, prefix string) (string, error) {
	var code string
	err := r.db.WithContext(ctx).Model(&CleaningTask{}).
		Where("code LIKE ?", prefix+"-%").
		Order("code DESC").
		Limit(1).
		Pluck("code", &code).Error
	return code, err
}

// Transition 在指定前置状态下更新任务状态，避免并发下出现非法流转。
func (r *Repository) Transition(ctx context.Context, id uint, from, to string, extra map[string]any) error {
	return r.TransitionTx(ctx, nil, id, from, to, extra)
}

// TransitionTx 在给定事务中执行状态流转，供验收模块与验收记录写入保持原子性。
//
// tx 可以为 nil，此时直接使用仓储自身的连接。
func (r *Repository) TransitionTx(ctx context.Context, tx *gorm.DB, id uint, from, to string, extra map[string]any) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	updates := map[string]any{
		"status":     to,
		"updated_at": time.Now(),
	}
	for key, value := range extra {
		updates[key] = value
	}
	result := db.WithContext(ctx).Model(&CleaningTask{}).
		Where("id = ? AND status = ?", id, from).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrStateConflict
	}
	return nil
}

// List 分页查询任务。
func (r *Repository) List(ctx context.Context, query ListQuery) ([]CleaningTask, int64, error) {
	query.Page.Normalize()
	var total int64
	if err := r.filtered(ctx, query).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	tasks := make([]CleaningTask, 0)
	err := r.filtered(ctx, query).
		Order("plan_start_date DESC, id DESC").
		Offset(query.Page.Offset()).
		Limit(query.Page.PageSize).
		Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *Repository) filtered(ctx context.Context, query ListQuery) *gorm.DB {
	tx := r.db.WithContext(ctx).Model(&CleaningTask{})
	if keyword := strings.ToLower(strings.TrimSpace(query.Keyword)); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where(
			"LOWER(code) LIKE ? OR LOWER(title) LIKE ? OR LOWER(team_name) LIKE ? OR LOWER(leader_name) LIKE ?",
			like, like, like, like,
		)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}
	if query.Priority != "" {
		tx = tx.Where("priority = ?", query.Priority)
	}
	if query.Source != "" {
		tx = tx.Where("source = ?", query.Source)
	}
	if query.PipeSegmentID > 0 {
		tx = tx.Where("pipe_segment_id = ?", query.PipeSegmentID)
	}
	if query.District != "" {
		subQuery := r.db.WithContext(ctx).Table(refx.TablePipeSegments).
			Select("id").
			Where("district = ?", query.District)
		tx = tx.Where("pipe_segment_id IN (?)", subQuery)
	}
	if query.PlanFrom != nil {
		tx = tx.Where("plan_start_date >= ?", query.PlanFrom.Time)
	}
	if query.PlanTo != nil {
		tx = tx.Where("plan_start_date <= ?", query.PlanTo.Time)
	}
	return tx
}

// HasRecords 任务下是否已经有清淤记录。
func (r *Repository) HasRecords(ctx context.Context, taskID uint) (bool, error) {
	return refx.HasRecordsForTask(ctx, r.db, taskID)
}

// HasAcceptance 任务下是否已经有验收记录。
func (r *Repository) HasAcceptance(ctx context.Context, taskID uint) (bool, error) {
	return refx.HasAcceptanceForTask(ctx, r.db, taskID)
}

// CountByStatus 统计各状态任务数量。
func (r *Repository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Status string
		Total  int64
	}
	rows := make([]row, 0)
	err := r.db.WithContext(ctx).Model(&CleaningTask{}).
		Select("status, COUNT(*) AS total").
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, item := range rows {
		result[item.Status] = item.Total
	}
	return result, nil
}
