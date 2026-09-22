package cleaningrecord

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/shared/refx"
)

// ErrNotFound 清淤记录不存在。
var ErrNotFound = errors.New("清淤记录不存在")

// ErrVersionConflict 记录已被其他人修改，当前版本已过期。
var ErrVersionConflict = errors.New("清淤记录版本已过期，请刷新后重试")

// Repository 清淤记录数据访问。
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

// Transaction 在一个事务内执行记录写入与版本留痕，保证两者同时成功或同时失败。
func (r *Repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// CreateInTx 在给定事务中新增记录。
func (r *Repository) CreateInTx(ctx context.Context, tx *gorm.DB, record *CleaningRecord) error {
	if record.Version == 0 {
		record.Version = 1
	}
	return tx.WithContext(ctx).Create(record).Error
}

// AddRevisionInTx 在给定事务中追加一条版本履历。
func (r *Repository) AddRevisionInTx(ctx context.Context, tx *gorm.DB, revision *CleaningRecordRevision) error {
	return tx.WithContext(ctx).Create(revision).Error
}

// UpdateInTx 在给定事务中按版本号乐观更新记录。
//
// WHERE 条件带上当前版本号：窗口关闭后即使绕过页面直接提交，
// 只要任务环节或验收引用在事务内复检不通过，整笔事务（含履历）都会回滚；
// 若期间记录已被另一笔提交改动，乐观条件不命中也会整体回滚。
func (r *Repository) UpdateInTx(ctx context.Context, tx *gorm.DB, current, next *CleaningRecord) error {
	updates := map[string]any{
		"cleaned_at":           next.CleanedAt,
		"length_m":             next.LengthM,
		"sludge_volume_m3":     next.SludgeVolumeM3,
		"water_volume_m3":      next.WaterVolumeM3,
		"personnel_count":      next.PersonnelCount,
		"method":               next.Method,
		"equipment":            next.Equipment,
		"weather":              next.Weather,
		"sludge_disposal_site": next.SludgeDisposalSite,
		"safety_measures":      next.SafetyMeasures,
		"problem_found":        next.ProblemFound,
		"recorder_name":        next.RecorderName,
		"remark":               next.Remark,
		"version":              next.Version,
		"updated_at":           time.Now(),
	}
	result := tx.WithContext(ctx).Model(&CleaningRecord{}).
		Where("id = ? AND version = ?", current.ID, current.Version).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVersionConflict
	}
	return nil
}

// DeleteInTx 在给定事务中物理删除记录及其全部版本履历。
func (r *Repository) DeleteInTx(ctx context.Context, tx *gorm.DB, id uint) error {
	result := tx.WithContext(ctx).Where("record_id = ?", id).Delete(&CleaningRecordRevision{})
	if result.Error != nil {
		return result.Error
	}
	result = tx.WithContext(ctx).Delete(&CleaningRecord{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete 物理删除记录（保留无事务的兼容入口）。
func (r *Repository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&CleaningRecord{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByID 按主键查询记录。
func (r *Repository) FindByID(ctx context.Context, id uint) (*CleaningRecord, error) {
	var record CleaningRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// TaskStatusInTx 查询任务当前状态（事务内复检修改窗口，防止绕过页面的并发提交）。
func (r *Repository) TaskStatusInTx(ctx context.Context, tx *gorm.DB, taskID uint) (string, error) {
	var status string
	err := tx.WithContext(ctx).Table(refx.TableCleaningTasks).
		Select("status").
		Where("id = ?", taskID).
		Scan(&status).Error
	return status, err
}

// HasAcceptanceInTx 事务内检查记录是否已被验收引用。
func (r *Repository) HasAcceptanceInTx(ctx context.Context, tx *gorm.DB, recordID uint) (bool, error) {
	var count int64
	err := tx.WithContext(ctx).Table(refx.TableAcceptanceRecords).
		Where("cleaning_record_id = ?", recordID).
		Count(&count).Error
	return count > 0, err
}

// ListRevisions 查询记录的全部版本履历（最新版本在前），并反序列化快照。
func (r *Repository) ListRevisions(ctx context.Context, recordID uint) ([]RevisionItem, error) {
	revisions := make([]CleaningRecordRevision, 0)
	err := r.db.WithContext(ctx).
		Where("record_id = ?", recordID).
		Order("version DESC, id DESC").
		Find(&revisions).Error
	if err != nil {
		return nil, err
	}
	items := make([]RevisionItem, 0, len(revisions))
	for i := range revisions {
		rev := revisions[i]
		item := RevisionItem{
			Version:      rev.Version,
			Action:       rev.Action,
			ChangedBy:    rev.ChangedBy,
			ChangeReason: rev.ChangeReason,
			ChangedAt:    rev.ChangedAt.Format(time.RFC3339),
		}
		if err := json.Unmarshal([]byte(rev.Snapshot), &item.Snapshot); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// marshalSnapshot 序列化版本快照。
func marshalSnapshot(snapshot VersionSnapshot) (string, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// MaxCodeWithPrefix 查询前缀下已使用的最大记录编号。
func (r *Repository) MaxCodeWithPrefix(ctx context.Context, prefix string) (string, error) {
	var code string
	err := r.db.WithContext(ctx).Model(&CleaningRecord{}).
		Where("code LIKE ?", prefix+"-%").
		Order("code DESC").
		Limit(1).
		Pluck("code", &code).Error
	return code, err
}

// List 分页查询清淤记录。
func (r *Repository) List(ctx context.Context, query ListQuery) ([]CleaningRecord, int64, error) {
	query.Page.Normalize()
	var total int64
	if err := r.filtered(ctx, query).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	records := make([]CleaningRecord, 0)
	err := r.filtered(ctx, query).
		Order("cleaned_at DESC, id DESC").
		Offset(query.Page.Offset()).
		Limit(query.Page.PageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *Repository) filtered(ctx context.Context, query ListQuery) *gorm.DB {
	tx := r.db.WithContext(ctx).Model(&CleaningRecord{})
	if keyword := strings.ToLower(strings.TrimSpace(query.Keyword)); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where(
			"LOWER(code) LIKE ? OR LOWER(equipment) LIKE ? OR LOWER(recorder_name) LIKE ? OR LOWER(sludge_disposal_site) LIKE ?",
			like, like, like, like,
		)
	}
	if query.TaskID > 0 {
		tx = tx.Where("task_id = ?", query.TaskID)
	}
	if query.SegmentID > 0 {
		subQuery := r.db.WithContext(ctx).Table(refx.TableCleaningTasks).
			Select("id").
			Where("pipe_segment_id = ?", query.SegmentID)
		tx = tx.Where("task_id IN (?)", subQuery)
	}
	if query.Method != "" {
		tx = tx.Where("method = ?", query.Method)
	}
	if query.Weather != "" {
		tx = tx.Where("weather = ?", query.Weather)
	}
	if query.DateFrom != nil {
		tx = tx.Where("cleaned_at >= ?", query.DateFrom.Time)
	}
	if query.DateTo != nil {
		tx = tx.Where("cleaned_at <= ?", query.DateTo.Time)
	}
	return tx
}

// HasAcceptance 记录是否已被验收引用。
func (r *Repository) HasAcceptance(ctx context.Context, recordID uint) (bool, error) {
	return refx.HasAcceptanceForRecord(ctx, r.db, recordID)
}

// TotalsByTask 汇总某个任务的清淤量。
func (r *Repository) TotalsByTask(ctx context.Context, taskID uint) (refx.RecordTotals, error) {
	return refx.TotalsByTaskID(ctx, r.db, taskID)
}
