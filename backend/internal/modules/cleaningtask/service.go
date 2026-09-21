package cleaningtask

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/option"
	"github.com/drainage/desilting/internal/shared/refx"
)

// SegmentGateway 管段台账对外提供的能力（由 pipesegment.Service 实现）。
type SegmentGateway interface {
	FindByID(ctx context.Context, id uint) (*pipesegment.PipeSegment, error)
	BriefsByIDs(ctx context.Context, ids []uint) (map[uint]pipesegment.Brief, error)
}

// Service 清淤任务业务逻辑。
type Service struct {
	repo     *Repository
	segments SegmentGateway
}

// NewService 构造服务。
func NewService(repo *Repository, segments SegmentGateway) *Service {
	return &Service{repo: repo, segments: segments}
}

// Create 登记清淤任务，任务编号按 日期 + 流水号 自动生成。
func (s *Service) Create(ctx context.Context, req SaveRequest) (*CleaningTask, error) {
	task := &CleaningTask{}
	if err := s.build(ctx, req, task); err != nil {
		return nil, err
	}

	for attempt := 0; attempt < 5; attempt++ {
		task.Code = s.nextCode(ctx, task.PlanStartDate)
		err := s.repo.Create(ctx, task)
		if err == nil {
			return task, nil
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.WrapInternal("新增清淤任务失败", err)
		}
	}
	return nil, httpx.Conflict("任务编号生成冲突，请稍后重试")
}

// Update 修改任务。已完工进入验收流程后不再允许改动计划信息。
func (s *Service) Update(ctx context.Context, id uint, req SaveRequest) (*CleaningTask, error) {
	task, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	if task.Status != StatusPending && task.Status != StatusInProgress {
		return nil, httpx.InvalidState(fmt.Sprintf(
			"任务当前状态为「%s」，只有待开工或清淤中的任务可以修改", StatusLabel(task.Status),
		))
	}

	hasRecords, err := s.repo.HasRecords(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("检查清淤记录失败", err)
	}
	if hasRecords && req.PipeSegmentID != task.PipeSegmentID {
		return nil, httpx.InvalidState("任务已录入清淤记录，不能再更换关联管段")
	}

	if err := s.build(ctx, req, task); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, task); err != nil {
		return nil, httpx.WrapInternal("修改清淤任务失败", err)
	}
	return task, nil
}

// Delete 删除任务。已产生清淤记录或验收记录的任务不允许删除，保证台账可追溯。
func (s *Service) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return notFound(err)
	}
	hasRecords, err := s.repo.HasRecords(ctx, id)
	if err != nil {
		return httpx.WrapInternal("检查清淤记录失败", err)
	}
	if hasRecords {
		return httpx.Conflict("该任务已录入清淤记录，无法删除")
	}
	hasAcceptance, err := s.repo.HasAcceptance(ctx, id)
	if err != nil {
		return httpx.WrapInternal("检查验收记录失败", err)
	}
	if hasAcceptance {
		return httpx.Conflict("该任务已存在验收记录，无法删除")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return notFound(err)
	}
	return nil
}

// FindByID 查询任务。
func (s *Service) FindByID(ctx context.Context, id uint) (*CleaningTask, error) {
	task, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	return task, nil
}

// List 分页查询任务，并批量补齐管段信息与清淤汇总。
func (s *Service) List(ctx context.Context, query ListQuery) ([]ListItem, int64, error) {
	tasks, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, httpx.WrapInternal("查询清淤任务失败", err)
	}
	if len(tasks) == 0 {
		return []ListItem{}, total, nil
	}

	segmentIDs := make([]uint, 0, len(tasks))
	taskIDs := make([]uint, 0, len(tasks))
	for i := range tasks {
		taskIDs = append(taskIDs, tasks[i].ID)
		segmentIDs = append(segmentIDs, tasks[i].PipeSegmentID)
	}

	briefs, err := s.segments.BriefsByIDs(ctx, segmentIDs)
	if err != nil {
		return nil, 0, err
	}
	totals, err := refx.TotalsByTaskIDs(ctx, s.repo.DB(), taskIDs)
	if err != nil {
		return nil, 0, httpx.WrapInternal("统计清淤量失败", err)
	}

	items := make([]ListItem, 0, len(tasks))
	for i := range tasks {
		task := tasks[i]
		item := ListItem{CleaningTask: task, RecordTotals: totals[task.ID]}
		if brief, ok := briefs[task.PipeSegmentID]; ok {
			item.Segment = &brief
		}
		items = append(items, item)
	}
	return items, total, nil
}

// Detail 任务详情。
func (s *Service) Detail(ctx context.Context, id uint) (*DetailResponse, error) {
	task, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &DetailResponse{Task: task, AllowedActions: AllowedActions(task.Status)}
	if briefs, err := s.segments.BriefsByIDs(ctx, []uint{task.PipeSegmentID}); err == nil {
		if brief, ok := briefs[task.PipeSegmentID]; ok {
			detail.Segment = &brief
		}
	} else {
		return nil, err
	}

	totals, err := refx.TotalsByTaskID(ctx, s.repo.DB(), id)
	if err != nil {
		return nil, httpx.WrapInternal("统计清淤量失败", err)
	}
	detail.RecordTotals = totals

	acceptance, err := refx.LatestAcceptanceForTask(ctx, s.repo.DB(), id)
	if err != nil {
		return nil, httpx.WrapInternal("查询验收结论失败", err)
	}
	detail.Acceptance = acceptance
	return detail, nil
}

// Start 开工：待开工 -> 清淤中。
func (s *Service) Start(ctx context.Context, id uint) (*CleaningTask, error) {
	task, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanTransition(task.Status, StatusInProgress) {
		return nil, TransitionError(task.Status, StatusInProgress)
	}
	now := time.Now()
	if err := s.repo.Transition(ctx, id, task.Status, StatusInProgress, map[string]any{"started_at": now}); err != nil {
		return nil, transitionFailure(err)
	}
	return s.FindByID(ctx, id)
}

// Complete 完工报验：清淤中 -> 待验收。必须至少有一条清淤记录。
func (s *Service) Complete(ctx context.Context, id uint) (*CleaningTask, error) {
	task, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanTransition(task.Status, StatusCompleted) {
		return nil, TransitionError(task.Status, StatusCompleted)
	}
	hasRecords, err := s.repo.HasRecords(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("检查清淤记录失败", err)
	}
	if !hasRecords {
		return nil, httpx.InvalidState("任务还没有清淤记录，请先录入清淤记录再提交完工报验")
	}
	now := time.Now()
	if err := s.repo.Transition(ctx, id, task.Status, StatusCompleted, map[string]any{"finished_at": now}); err != nil {
		return nil, transitionFailure(err)
	}
	return s.FindByID(ctx, id)
}

// Cancel 取消任务：仅待开工、清淤中可取消。
func (s *Service) Cancel(ctx context.Context, id uint, reason string) (*CleaningTask, error) {
	task, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanTransition(task.Status, StatusCancelled) {
		return nil, TransitionError(task.Status, StatusCancelled)
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return nil, httpx.Validation("取消原因不能为空")
	}
	if err := s.repo.Transition(ctx, id, task.Status, StatusCancelled, map[string]any{"cancel_reason": trimmed}); err != nil {
		return nil, transitionFailure(err)
	}
	return s.FindByID(ctx, id)
}

// EnsureStarted 录入清淤记录时把任务推进到清淤中（幂等）。
func (s *Service) EnsureStarted(ctx context.Context, id uint) (*CleaningTask, error) {
	task, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	switch task.Status {
	case StatusPending:
		now := time.Now()
		if err := s.repo.Transition(ctx, id, StatusPending, StatusInProgress, map[string]any{"started_at": now}); err != nil {
			return nil, transitionFailure(err)
		}
		return s.FindByID(ctx, id)
	case StatusInProgress:
		return task, nil
	default:
		return nil, httpx.InvalidState(fmt.Sprintf(
			"任务当前状态为「%s」，不能录入清淤记录", StatusLabel(task.Status),
		))
	}
}

// TransitionInTx 在给定事务中推进任务状态，供验收模块联动更新使用。
//
// 状态流转规则仍由本模块统一维护（CanTransition），验收模块只负责提供目标状态。
func (s *Service) TransitionInTx(ctx context.Context, tx *gorm.DB, id uint, from, to string, extra map[string]any) error {
	if !CanTransition(from, to) {
		return TransitionError(from, to)
	}
	if err := s.repo.TransitionTx(ctx, tx, id, from, to, extra); err != nil {
		return transitionFailure(err)
	}
	return nil
}

// CountByStatus 统计各状态任务数（供看板使用）。
func (s *Service) CountByStatus(ctx context.Context) (map[string]int64, error) {
	counts, err := s.repo.CountByStatus(ctx)
	if err != nil {
		return nil, httpx.WrapInternal("统计任务状态失败", err)
	}
	return counts, nil
}

// AllowedActions 根据当前状态给出前端可执行的下一步操作。
func AllowedActions(status string) []string {
	switch status {
	case StatusPending:
		return []string{ActionStart, ActionEdit, ActionCancel}
	case StatusInProgress:
		return []string{ActionComplete, ActionEdit, ActionCancel}
	case StatusCompleted:
		return []string{ActionAccept}
	default:
		return []string{}
	}
}

// build 校验并写入任务字段。
func (s *Service) build(ctx context.Context, req SaveRequest, target *CleaningTask) error {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return httpx.Validation("任务标题不能为空")
	}
	if req.PipeSegmentID == 0 {
		return httpx.Validation("请选择关联管段")
	}
	if _, err := s.segments.FindByID(ctx, req.PipeSegmentID); err != nil {
		return err
	}

	priority := strings.TrimSpace(req.Priority)
	if priority == "" {
		priority = PriorityNormal
	}
	if !option.Has(PriorityOptions(), priority) {
		return httpx.Validation(fmt.Sprintf("优先级只能是：%s", option.Labels(PriorityOptions())))
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = SourcePlan
	}
	if !option.Has(SourceOptions(), source) {
		return httpx.Validation(fmt.Sprintf("任务来源只能是：%s", option.Labels(SourceOptions())))
	}

	method := strings.TrimSpace(req.Method)
	if method != "" && !option.Has(MethodOptions(), method) {
		return httpx.Validation(fmt.Sprintf("清淤方式只能是：%s", option.Labels(MethodOptions())))
	}

	if req.PlanStartDate.IsZero() {
		return httpx.Validation("计划开始日期不能为空")
	}
	if req.PlanEndDate.IsZero() {
		return httpx.Validation("计划完成日期不能为空")
	}
	if req.PlanEndDate.Before(req.PlanStartDate) {
		return httpx.Validation("计划完成日期不能早于计划开始日期")
	}

	phone := strings.TrimSpace(req.LeaderPhone)
	if phone != "" && !isPhoneLike(phone) {
		return httpx.Validation("联系电话只能包含数字、空格、加号和横线，长度 6-32 位")
	}

	target.Title = title
	target.PipeSegmentID = req.PipeSegmentID
	target.Priority = priority
	target.Source = source
	target.Method = method
	target.PlanStartDate = req.PlanStartDate
	target.PlanEndDate = req.PlanEndDate
	target.TeamName = strings.TrimSpace(req.TeamName)
	target.LeaderName = strings.TrimSpace(req.LeaderName)
	target.LeaderPhone = phone
	target.Description = strings.TrimSpace(req.Description)
	if target.Status == "" {
		target.Status = StatusPending
	}
	return nil
}

// nextCode 生成形如 QX20260914-0001 的任务编号。
func (s *Service) nextCode(ctx context.Context, planStart date.Date) string {
	prefix := "QX" + planStart.Format("20060102")
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

func isPhoneLike(value string) bool {
	if len([]rune(value)) < 6 || len([]rune(value)) > 32 {
		return false
	}
	for _, r := range value {
		if unicode.IsDigit(r) || r == ' ' || r == '-' || r == '+' {
			continue
		}
		return false
	}
	return true
}

func transitionFailure(err error) error {
	if errors.Is(err, ErrStateConflict) {
		return httpx.InvalidState("任务状态已被其他操作变更，请刷新后重试")
	}
	return httpx.WrapInternal("更新任务状态失败", err)
}

func notFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound("清淤任务不存在")
	}
	return httpx.WrapInternal("查询清淤任务失败", err)
}
