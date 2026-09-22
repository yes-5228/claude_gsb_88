package cleaningrecord

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/shared/refx"
)

// ErrNotFound 清淤记录不存在。
var ErrNotFound = errors.New("清淤记录不存在")

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

// Transaction 在事务中执行写入，保证修改与留痕同时成功或同时失败。
func (r *Repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// Create 新增记录。
func (r *Repository) Create(ctx context.Context, record *CleaningRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// SaveInTx 在给定事务中保存记录全部字段。
func (r *Repository) SaveInTx(ctx context.Context, tx *gorm.DB, record *CleaningRecord) error {
	record.UpdatedAt = time.Now()
	return tx.WithContext(ctx).Save(record).Error
}

// DeleteWithRevisionsInTx 在给定事务中删除记录及其全部留痕。
func (r *Repository) DeleteWithRevisionsInTx(ctx context.Context, tx *gorm.DB, id uint) error {
	if err := tx.WithContext(ctx).Where("record_id = ?", id).Delete(&CleaningRecordRevision{}).Error; err != nil {
		return err
	}
	result := tx.WithContext(ctx).Delete(&CleaningRecord{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateRevisionInTx 在给定事务中写入一条修改留痕。
func (r *Repository) CreateRevisionInTx(ctx context.Context, tx *gorm.DB, revision *CleaningRecordRevision) error {
	return tx.WithContext(ctx).Create(revision).Error
}

// MaxRevisionVersion 查询记录已留痕的最大版本号，没有留痕时返回 0。
//
// 传入事务句柄可保证版本号判断与留痕写入在同一个事务内，避免并发写出重复版本。
func (r *Repository) MaxRevisionVersion(ctx context.Context, db *gorm.DB, recordID uint) (int, error) {
	var maxVersion int
	err := db.WithContext(ctx).Model(&CleaningRecordRevision{}).
		Where("record_id = ?", recordID).
		Pluck("COALESCE(MAX(version), 0)", &maxVersion).Error
	return maxVersion, err
}

// ListRevisions 查询记录的全部留痕（按版本号升序）。
func (r *Repository) ListRevisions(ctx context.Context, recordID uint) ([]CleaningRecordRevision, error) {
	revisions := make([]CleaningRecordRevision, 0)
	err := r.db.WithContext(ctx).
		Where("record_id = ?", recordID).
		Order("version ASC").
		Find(&revisions).Error
	return revisions, err
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

// TotalsByTask 汇总某个任务的清淤量。
func (r *Repository) TotalsByTask(ctx context.Context, taskID uint) (refx.RecordTotals, error) {
	return refx.TotalsByTaskID(ctx, r.db, taskID)
}
