package cleaningrecord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
		err := s.repo.Create(ctx, record)
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
// 修改窗口由任务所处环节决定：任务处于「待开工」「清淤中」时窗口开放，
// 完工报验（待验收）后窗口关闭；已被验收引用的记录只能查看历史版本。
// 窗口复查、修改前快照留痕与数值更新在同一个事务内完成：
// 数值变了就一定有对应留痕，留痕写不进去数值也不会变；
// 即使绕过页面直接提交，窗口关闭后也会被这里拦下。
func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*CleaningRecord, error) {
	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	if req.TaskID != record.TaskID {
		return nil, httpx.InvalidState("清淤记录不支持更换所属任务，如需调整请删除后重新录入")
	}
	if err := validate(req.SaveRequest); err != nil {
		return nil, err
	}
	editorName := strings.TrimSpace(req.EditorName)
	if editorName == "" {
		return nil, httpx.Validation("修改人不能为空")
	}
	changeReason := strings.TrimSpace(req.ChangeReason)
	if changeReason == "" {
		return nil, httpx.Validation("修改原因不能为空")
	}

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.ensureEditable(ctx, tx, record); err != nil {
			return err
		}
		version, err := s.repo.MaxRevisionVersion(ctx, tx, id)
		if err != nil {
			return httpx.WrapInternal("查询修改留痕失败", err)
		}
		revision := snapshotOf(record, version+1, editorName, changeReason)
		if err := s.repo.CreateRevisionInTx(ctx, tx, revision); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return httpx.Conflict("该清淤记录正在被其他人修改，请刷新后重试")
			}
			return httpx.WrapInternal("写入修改留痕失败", err)
		}
		apply(req.SaveRequest, record)
		if err := s.repo.SaveInTx(ctx, tx, record); err != nil {
			return httpx.WrapInternal("修改清淤记录失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

// Delete 删除清淤记录，留痕随记录在同一事务内一并清除。
func (s *Service) Delete(ctx context.Context, id uint) error {
	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return notFound(err)
	}
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.ensureEditable(ctx, tx, record); err != nil {
			return err
		}
		return s.repo.DeleteWithRevisionsInTx(ctx, tx, id)
	})
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			return err
		}
		return notFound(err)
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

// TotalsByTask 汇总任务下的清淤量（供验收模块判断清淤成果）。
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

// Detail 记录详情，附带修改窗口状态与当前版本号。
func (s *Service) Detail(ctx context.Context, id uint) (*DetailResponse, error) {
	record, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	briefs, err := refx.TaskBriefsByIDs(ctx, s.repo.DB(), []uint{record.TaskID})
	if err != nil {
		return nil, httpx.WrapInternal("查询任务信息失败", err)
	}
	detail := &DetailResponse{Record: record}
	if brief, ok := briefs[record.TaskID]; ok {
		detail.Task = &brief
	}
	window, err := editWindowOf(ctx, s.repo.DB(), record)
	if err != nil {
		return nil, err
	}
	detail.EditWindow = window
	version, err := s.repo.MaxRevisionVersion(ctx, s.repo.DB(), id)
	if err != nil {
		return nil, httpx.WrapInternal("查询修改留痕失败", err)
	}
	detail.CurrentVersion = version + 1
	return detail, nil
}

// Versions 查询记录的全部版本（含当前版本），按版本号升序返回，详情页可对比任意两个版本。
func (s *Service) Versions(ctx context.Context, id uint) (*VersionListResponse, error) {
	record, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	revisions, err := s.repo.ListRevisions(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("查询修改留痕失败", err)
	}
	versions := make([]RecordVersion, 0, len(revisions)+1)
	for i := range revisions {
		versions = append(versions, versionFromRevision(&revisions[i]))
	}
	versions = append(versions, versionFromRecord(record, len(revisions)+1))
	return &VersionListResponse{CurrentVersion: len(revisions) + 1, Versions: versions}, nil
}

// ensureEditable 在事务内复查修改窗口，窗口关闭时阻断调整。
func (s *Service) ensureEditable(ctx context.Context, tx *gorm.DB, record *CleaningRecord) error {
	window, err := editWindowOf(ctx, tx, record)
	if err != nil {
		return err
	}
	if !window.Open {
		return httpx.InvalidState(window.Reason)
	}
	return nil
}

// editWindowOf 计算记录的修改窗口状态：窗口起止由任务所处环节决定，
// 任务完工报验后窗口关闭；被验收引用的记录只能查看历史版本。
func editWindowOf(ctx context.Context, db *gorm.DB, record *CleaningRecord) (EditWindow, error) {
	status, err := refx.TaskStatusByID(ctx, db, record.TaskID)
	if err != nil {
		return EditWindow{}, httpx.WrapInternal("查询任务状态失败", err)
	}
	if status == "" {
		return EditWindow{}, httpx.NotFound("清淤任务不存在")
	}
	if !editable(status) {
		return EditWindow{Open: false, Reason: fmt.Sprintf(
			"任务当前状态为「%s」，修改窗口已关闭，清淤记录不能再调整", cleaningtask.StatusLabel(status),
		)}, nil
	}
	referenced, err := refx.HasAcceptanceForRecord(ctx, db, record.ID)
	if err != nil {
		return EditWindow{}, httpx.WrapInternal("检查验收引用失败", err)
	}
	if referenced {
		return EditWindow{Open: false, Reason: "该清淤记录已被验收记录引用，只能查看历史版本，不能再调整"}, nil
	}
	return EditWindow{Open: true}, nil
}

// snapshotOf 把记录当前（修改前）的数值保存为一个历史版本。
func snapshotOf(record *CleaningRecord, version int, editorName, changeReason string) *CleaningRecordRevision {
	return &CleaningRecordRevision{
		RecordID:           record.ID,
		Version:            version,
		CleanedAt:          record.CleanedAt,
		LengthM:            record.LengthM,
		SludgeVolumeM3:     record.SludgeVolumeM3,
		WaterVolumeM3:      record.WaterVolumeM3,
		PersonnelCount:     record.PersonnelCount,
		Method:             record.Method,
		Equipment:          record.Equipment,
		Weather:            record.Weather,
		SludgeDisposalSite: record.SludgeDisposalSite,
		SafetyMeasures:     record.SafetyMeasures,
		ProblemFound:       record.ProblemFound,
		RecorderName:       record.RecorderName,
		Remark:             record.Remark,
		EditorName:         editorName,
		ChangeReason:       changeReason,
	}
}

// versionFromRevision 把留痕快照转成版本视图。
func versionFromRevision(revision *CleaningRecordRevision) RecordVersion {
	changedAt := revision.CreatedAt
	return RecordVersion{
		Version:            revision.Version,
		Current:            false,
		EditorName:         revision.EditorName,
		ChangeReason:       revision.ChangeReason,
		ChangedAt:          &changedAt,
		CleanedAt:          revision.CleanedAt,
		LengthM:            revision.LengthM,
		SludgeVolumeM3:     revision.SludgeVolumeM3,
		WaterVolumeM3:      revision.WaterVolumeM3,
		PersonnelCount:     revision.PersonnelCount,
		Method:             revision.Method,
		Equipment:          revision.Equipment,
		Weather:            revision.Weather,
		SludgeDisposalSite: revision.SludgeDisposalSite,
		SafetyMeasures:     revision.SafetyMeasures,
		ProblemFound:       revision.ProblemFound,
		RecorderName:       revision.RecorderName,
		Remark:             revision.Remark,
	}
}

// versionFromRecord 把记录当前行转成版本视图（最新版本）。
func versionFromRecord(record *CleaningRecord, version int) RecordVersion {
	return RecordVersion{
		Version:            version,
		Current:            true,
		CleanedAt:          record.CleanedAt,
		LengthM:            record.LengthM,
		SludgeVolumeM3:     record.SludgeVolumeM3,
		WaterVolumeM3:      record.WaterVolumeM3,
		PersonnelCount:     record.PersonnelCount,
		Method:             record.Method,
		Equipment:          record.Equipment,
		Weather:            record.Weather,
		SludgeDisposalSite: record.SludgeDisposalSite,
		SafetyMeasures:     record.SafetyMeasures,
		ProblemFound:       record.ProblemFound,
		RecorderName:       record.RecorderName,
		Remark:             record.Remark,
	}
}

// editable 判断任务是否处于可以增删改清淤记录的状态。
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
