package cleaningrecord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/option"
	"github.com/drainage/desilting/internal/shared/refx"
)

// TaskGateway 清淤任务模块对外提供的能力（由 cleaningtask.Service 实现）。
type TaskGateway interface {
	FindByID(ctx context.Context, id uint) (*cleaningtask.CleaningTask, error)
	EnsureStarted(ctx context.Context, id uint) (*cleaningtask.CleaningTask, error)
}

// createReason 首次录入版本的履历原因。
const createReason = "首次录入"

// Service 清淤记录业务逻辑。
type Service struct {
	repo  *Repository
	tasks TaskGateway
}

// NewService 构造服务。
func NewService(repo *Repository, tasks TaskGateway) *Service {
	return &Service{repo: repo, tasks: tasks}
}

// Create 录入清淤记录。首次录入时会把任务从"待开工"推进到"清淤中"。
//
// 记录主表与第 1 版履历在同一个事务内写入：不可能出现记录已存在却没有履历，
// 也不会出现履历存在但记录没有写入。
func (s *Service) Create(ctx context.Context, req SaveRequest) (*CleaningRecord, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	// 先确认任务可录入（同时完成状态推进），避免写入脏数据。
	if _, err := s.tasks.EnsureStarted(ctx, req.TaskID); err != nil {
		return nil, err
	}

	record := &CleaningRecord{}
	apply(req, record)

	for attempt := 0; attempt < 5; attempt++ {
		record.Code = s.nextCode(ctx, record.CleanedAt)
		err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
			if err := s.repo.CreateInTx(ctx, tx, record); err != nil {
				return err
			}
			return s.addRevision(ctx, tx, record, RevisionActionCreate, record.RecorderName, createReason)
		})
		if err == nil {
			return record, nil
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.WrapInternal("录入清淤记录失败", err)
		}
	}
	return nil, httpx.Conflict("记录编号生成冲突，请稍后重试")
}

// Update 修改清淤记录。
//
// 修改窗口由任务所处环节决定（待开工 / 清淤中），且记录未被验收引用。
// 窗口控制在事务内复检：即使绕过页面直接提交，或在提交瞬间任务被报验 / 验收，
// 也会被拦住。数值更新与履历留痕在同一事务内提交，任一失败全部回滚。
func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*CleaningRecord, error) {
	if err := validate(req.SaveRequest); err != nil {
		return nil, err
	}
	modifier := strings.TrimSpace(req.ModifierName)
	if modifier == "" {
		return nil, httpx.Validation("修改人不能为空")
	}
	reason := strings.TrimSpace(req.ChangeReason)
	if reason == "" {
		return nil, httpx.Validation("修改原因不能为空")
	}

	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	if req.TaskID != record.TaskID {
		return nil, httpx.InvalidState("清淤记录不支持更换所属任务，如需调整请删除后重新录入")
	}
	// 窗口预检（事务内还会复检一次），尽早给出明确提示。
	if err := s.ensureEditable(ctx, nil, record); err != nil {
		return nil, err
	}

	next := *record
	apply(req.SaveRequest, &next)
	// 没有任何字段变化时不产生新版本，避免无意义的留痕与版本号膨胀。
	if snapshotOf(&next) == snapshotOf(record) {
		return nil, httpx.Validation("本次提交与当前版本内容一致，没有需要保存的改动")
	}
	// 履历保存的是「修改后」版本，先把版本号递增；随后履历插入与主表乐观更新
	// 必须带相同版本号，唯一索引 (record_id, version) 会挡住并发的第二笔修改。
	next.Version = record.Version + 1

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// 事务内再次复检窗口与引用，拦住并发报验 / 验收、绕过页面的提交。
		if err := s.ensureEditable(ctx, tx, record); err != nil {
			return err
		}
		if err := s.addRevision(ctx, tx, &next, RevisionActionUpdate, modifier, reason); err != nil {
			return err
		}
		return s.repo.UpdateInTx(ctx, tx, record, &next)
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrVersionConflict):
			return nil, httpx.Conflict("该记录刚被其他人修改过，请刷新页面获取最新版本后再提交")
		case errors.Is(err, ErrNotFound):
			return nil, httpx.NotFound("清淤记录不存在")
		default:
			var appErr *httpx.AppError
			if errors.As(err, &appErr) {
				return nil, err
			}
			return nil, httpx.WrapInternal("修改清淤记录失败", err)
		}
	}
	return &next, nil
}

// Delete 删除清淤记录。窗口规则与修改一致，记录与履历在同一事务内删除。
func (s *Service) Delete(ctx context.Context, id uint) error {
	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return notFound(err)
	}
	if err := s.ensureEditable(ctx, nil, record); err != nil {
		return err
	}
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.ensureEditable(ctx, tx, record); err != nil {
			return err
		}
		return s.repo.DeleteInTx(ctx, tx, id)
	})
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			return err
		}
		if errors.Is(err, ErrNotFound) {
			return httpx.NotFound("清淤记录不存在")
		}
		return httpx.WrapInternal("删除清淤记录失败", err)
	}
	return nil
}

// FindByID 查询清淤记录。
func (s *Service) FindByID(ctx context.Context, id uint) (*CleaningRecord, error) {
	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	return record, nil
}

// Revisions 查询记录的版本履历（最新版本在前），供详情页对比任意两个版本。
func (s *Service) Revisions(ctx context.Context, id uint) (*RevisionListResponse, error) {
	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	items, err := s.repo.ListRevisions(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("查询清淤记录版本履历失败", err)
	}
	return &RevisionListResponse{RecordID: id, Code: record.Code, Items: items}, nil
}

// TotalsByTask 汇总任务下的清淤量（供验收模块判断清淤成果）。
//
// 只聚合主表（始终为最新版本），履历表不参与统计，因此统计口径天然为最新版本。
func (s *Service) TotalsByTask(ctx context.Context, taskID uint) (refx.RecordTotals, error) {
	totals, err := s.repo.TotalsByTask(ctx, taskID)
	if err != nil {
		return refx.RecordTotals{}, httpx.WrapInternal("统计清淤量失败", err)
	}
	return totals, nil
}

// List 分页查询清淤记录，并补齐所属任务与管段信息。
func (s *Service) List(ctx context.Context, query ListQuery) ([]ListItem, int64, error) {
	records, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, httpx.WrapInternal("查询清淤记录失败", err)
	}
	if len(records) == 0 {
		return []ListItem{}, total, nil
	}

	taskIDs := make([]uint, 0, len(records))
	for i := range records {
		taskIDs = append(taskIDs, records[i].TaskID)
	}
	briefs, err := refx.TaskBriefsByIDs(ctx, s.repo.DB(), taskIDs)
	if err != nil {
		return nil, 0, httpx.WrapInternal("查询任务信息失败", err)
	}

	items := make([]ListItem, 0, len(records))
	for i := range records {
		record := records[i]
		item := ListItem{CleaningRecord: record}
		if brief, ok := briefs[record.TaskID]; ok {
			item.Task = &brief
		}
		items = append(items, item)
	}
	return items, total, nil
}

// Detail 记录详情，附带修改窗口状态（供前端控制编辑入口）与版本履历概况。
func (s *Service) Detail(ctx context.Context, id uint) (*DetailResponse, error) {
	record, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	briefs, err := refx.TaskBriefsByIDs(ctx, s.repo.DB(), []uint{record.TaskID})
	if err != nil {
		return nil, httpx.WrapInternal("查询任务信息失败", err)
	}
	detail := &DetailResponse{Record: record, RevisionCount: record.Version, Editable: true}
	if brief, ok := briefs[record.TaskID]; ok {
		detail.Task = &brief
	}
	referenced, err := s.repo.HasAcceptance(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("检查验收引用失败", err)
	}
	if referenced {
		detail.Editable = false
		detail.LockedReason = "该记录已被验收记录引用，只能查看历史版本，不能继续修改"
	} else if task, err := s.tasks.FindByID(ctx, record.TaskID); err == nil && !editable(task.Status) {
		detail.Editable = false
		detail.LockedReason = fmt.Sprintf(
			"任务当前环节为「%s」，修改窗口已关闭，只能查看历史版本", cleaningtask.StatusLabel(task.Status),
		)
	}
	return detail, nil
}

// addRevision 在给定事务中为记录追加一个版本。
func (s *Service) addRevision(
	ctx context.Context,
	tx *gorm.DB,
	record *CleaningRecord,
	action string,
	changedBy string,
	reason string,
) error {
	snapshot, err := marshalSnapshot(snapshotOf(record))
	if err != nil {
		return httpx.WrapInternal("序列化版本快照失败", err)
	}
	now := time.Now()
	revision := &CleaningRecordRevision{
		RecordID:     record.ID,
		Version:      record.Version,
		Action:       action,
		Snapshot:     snapshot,
		ChangedBy:    strings.TrimSpace(changedBy),
		ChangeReason: reason,
		ChangedAt:    now,
	}
	return s.repo.AddRevisionInTx(ctx, tx, revision)
}

// ensureEditable 校验记录是否处于可修改 / 可删除的窗口内。
//
// tx 非空时在事务内复检（直接查任务状态与验收引用），
// 用于拦住「页面打开后任务被报验 / 验收」或绕过页面提交的并发情况。
func (s *Service) ensureEditable(ctx context.Context, tx *gorm.DB, record *CleaningRecord) error {
	if tx != nil {
		referenced, err := s.repo.HasAcceptanceInTx(ctx, tx, record.ID)
		if err != nil {
			return httpx.WrapInternal("检查验收引用失败", err)
		}
		if referenced {
			return httpx.InvalidState("该清淤记录已被验收记录引用，只能查看历史版本，不能继续修改")
		}
		status, err := s.repo.TaskStatusInTx(ctx, tx, record.TaskID)
		if err != nil {
			return httpx.WrapInternal("查询任务环节失败", err)
		}
		if !editable(status) {
			return httpx.InvalidState(fmt.Sprintf(
				"任务当前环节为「%s」，修改窗口已关闭，清淤记录不能继续调整", cleaningtask.StatusLabel(status),
			))
		}
		return nil
	}

	referenced, err := s.repo.HasAcceptance(ctx, record.ID)
	if err != nil {
		return httpx.WrapInternal("检查验收引用失败", err)
	}
	if referenced {
		return httpx.InvalidState("该清淤记录已被验收记录引用，只能查看历史版本，不能继续修改")
	}
	task, err := s.tasks.FindByID(ctx, record.TaskID)
	if err != nil {
		return err
	}
	if !editable(task.Status) {
		return httpx.InvalidState(fmt.Sprintf(
			"任务当前环节为「%s」，修改窗口已关闭，清淤记录不能继续调整", cleaningtask.StatusLabel(task.Status),
		))
	}
	return nil
}

// editable 判断任务是否处于可以增删改清淤记录的环节（即修改窗口开放）。
func editable(taskStatus string) bool {
	return taskStatus == cleaningtask.StatusPending || taskStatus == cleaningtask.StatusInProgress
}

func validate(req SaveRequest) error {
	if req.CleanedAt.IsZero() {
		return httpx.Validation("清淤日期不能为空")
	}
	if req.CleanedAt.After(date.Today()) {
		return httpx.Validation("清淤日期不能晚于今天")
	}
	method := strings.TrimSpace(req.Method)
	if method != "" && !option.Has(cleaningtask.MethodOptions(), method) {
		return httpx.Validation(fmt.Sprintf("清淤方式只能是：%s", option.Labels(cleaningtask.MethodOptions())))
	}
	weather := strings.TrimSpace(req.Weather)
	if weather != "" && !option.Has(WeatherOptions(), weather) {
		return httpx.Validation(fmt.Sprintf("天气只能是：%s", option.Labels(WeatherOptions())))
	}
	if strings.TrimSpace(req.RecorderName) == "" {
		return httpx.Validation("记录人不能为空")
	}
	return nil
}

func apply(req SaveRequest, target *CleaningRecord) {
	target.TaskID = req.TaskID
	target.CleanedAt = req.CleanedAt
	target.LengthM = req.LengthM
	target.SludgeVolumeM3 = req.SludgeVolumeM3
	target.WaterVolumeM3 = req.WaterVolumeM3
	target.PersonnelCount = req.PersonnelCount
	target.Method = strings.TrimSpace(req.Method)
	target.Equipment = strings.TrimSpace(req.Equipment)
	target.Weather = strings.TrimSpace(req.Weather)
	target.SludgeDisposalSite = strings.TrimSpace(req.SludgeDisposalSite)
	target.SafetyMeasures = strings.TrimSpace(req.SafetyMeasures)
	target.ProblemFound = strings.TrimSpace(req.ProblemFound)
	target.RecorderName = strings.TrimSpace(req.RecorderName)
	target.Remark = strings.TrimSpace(req.Remark)
}

// nextCode 生成形如 QJ20260914-0001 的记录编号。
func (s *Service) nextCode(ctx context.Context, cleanedAt date.Date) string {
	prefix := "QJ" + cleanedAt.Format("20060102")
	sequence := 1
	if latest, err := s.repo.MaxCodeWithPrefix(ctx, prefix); err == nil && latest != "" {
		if idx := strings.LastIndex(latest, "-"); idx >= 0 {
			if parsed, err := strconv.Atoi(latest[idx+1:]); err == nil {
				sequence = parsed + 1
			}
		}
	}
	return fmt.Sprintf("%s-%04d", prefix, sequence)
}

func notFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound("清淤记录不存在")
	}
	return httpx.WrapInternal("查询清淤记录失败", err)
}
