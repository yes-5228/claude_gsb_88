package pipesegment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/option"
	"github.com/drainage/desilting/internal/shared/refx"
)

// Service 管段台账业务逻辑。
type Service struct {
	repo *Repository
}

// NewService 构造服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create 新增管段。
func (s *Service) Create(ctx context.Context, req SaveRequest) (*PipeSegment, error) {
	segment := &PipeSegment{}
	if err := applyRequest(req, segment, true); err != nil {
		return nil, err
	}

	exists, err := s.repo.ExistsByCode(ctx, segment.Code, 0)
	if err != nil {
		return nil, httpx.WrapInternal("校验管段编号失败", err)
	}
	if exists {
		return nil, httpx.Conflict(fmt.Sprintf("管段编号 %s 已存在", segment.Code))
	}

	if err := s.repo.Create(ctx, segment); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.Conflict(fmt.Sprintf("管段编号 %s 已存在", segment.Code))
		}
		return nil, httpx.WrapInternal("新增管段失败", err)
	}
	return segment, nil
}

// Update 修改管段档案。
func (s *Service) Update(ctx context.Context, id uint, req SaveRequest) (*PipeSegment, error) {
	segment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	if err := applyRequest(req, segment, false); err != nil {
		return nil, err
	}

	exists, err := s.repo.ExistsByCode(ctx, segment.Code, id)
	if err != nil {
		return nil, httpx.WrapInternal("校验管段编号失败", err)
	}
	if exists {
		return nil, httpx.Conflict(fmt.Sprintf("管段编号 %s 已被其他管段使用", segment.Code))
	}

	if err := s.repo.Save(ctx, segment); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.Conflict(fmt.Sprintf("管段编号 %s 已被其他管段使用", segment.Code))
		}
		return nil, httpx.WrapInternal("修改管段失败", err)
	}
	return segment, nil
}

// Delete 删除管段。已被清淤任务引用的管段不允许删除，避免出现孤儿任务。
func (s *Service) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return notFound(err)
	}
	referenced, err := refx.HasTasksForSegment(ctx, s.repo.DB(), id)
	if err != nil {
		return httpx.WrapInternal("检查管段引用失败", err)
	}
	if referenced {
		return httpx.Conflict("该管段已存在清淤任务，无法删除；如需停用请把运行状态改为已淤堵")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return notFound(err)
	}
	return nil
}

// FindByID 查询单个管段。
func (s *Service) FindByID(ctx context.Context, id uint) (*PipeSegment, error) {
	segment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFound(err)
	}
	return segment, nil
}

// BriefsByIDs 批量查询管段精简信息（供其他模块使用）。
func (s *Service) BriefsByIDs(ctx context.Context, ids []uint) (map[uint]Brief, error) {
	briefs, err := s.repo.BriefsByIDs(ctx, ids)
	if err != nil {
		return nil, httpx.WrapInternal("查询管段信息失败", err)
	}
	return briefs, nil
}

// MarkCleaned 验收合格后更新管段清淤统计（供验收模块调用；tx 可以为 nil）。
func (s *Service) MarkCleaned(ctx context.Context, tx *gorm.DB, segmentID uint, cleanedAt date.Date) error {
	if err := s.repo.MarkCleaned(ctx, tx, segmentID, cleanedAt); err != nil {
		return httpx.WrapInternal("更新管段清淤统计失败", err)
	}
	return nil
}

// List 分页查询管段台账。
func (s *Service) List(ctx context.Context, query ListQuery) ([]PipeSegment, int64, error) {
	items, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, httpx.WrapInternal("查询管段列表失败", err)
	}
	return items, total, nil
}

// Detail 查询管段详情，附带任务统计与最近任务。
func (s *Service) Detail(ctx context.Context, id uint) (*DetailResponse, error) {
	segment, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	stats, err := s.repo.TaskStats(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("统计管段任务失败", err)
	}
	recent, err := s.repo.RecentTasks(ctx, id, 5)
	if err != nil {
		return nil, httpx.WrapInternal("查询管段任务失败", err)
	}
	return &DetailResponse{Segment: segment, TaskStats: stats, RecentTasks: recent}, nil
}

// History 查询管段的清淤履历（任务 + 清淤量 + 验收结论）。
func (s *Service) History(ctx context.Context, id uint) ([]refx.HistoryItem, error) {
	if _, err := s.FindByID(ctx, id); err != nil {
		return nil, err
	}
	items, err := s.repo.History(ctx, id)
	if err != nil {
		return nil, httpx.WrapInternal("查询管段清淤履历失败", err)
	}
	return items, nil
}

// Options 下拉选项。
func (s *Service) Options(ctx context.Context, keyword string) (*OptionsResponse, error) {
	items, err := s.repo.Search(ctx, keyword, 100)
	if err != nil {
		return nil, httpx.WrapInternal("查询管段选项失败", err)
	}
	districts, err := s.repo.Districts(ctx)
	if err != nil {
		return nil, httpx.WrapInternal("查询片区失败", err)
	}
	return &OptionsResponse{Items: items, Districts: districts}, nil
}

// applyRequest 把请求体写入目标对象并做枚举校验。
//
// isCreate 为 true 时要求状态必填（缺省取正常），为 false 时状态可选、不传则保持原值。
func applyRequest(req SaveRequest, target *PipeSegment, isCreate bool) error {
	pipeType := strings.TrimSpace(req.PipeType)
	if !option.Has(PipeTypeOptions(), pipeType) {
		return httpx.Validation(fmt.Sprintf("管段类型只能是：%s", option.Labels(PipeTypeOptions())))
	}

	status := strings.TrimSpace(req.Status)
	if status != "" {
		if !option.Has(StatusOptions(), status) {
			return httpx.Validation(fmt.Sprintf("运行状态只能是：%s", option.Labels(StatusOptions())))
		}
	} else if isCreate || target.Status == "" {
		status = StatusNormal
	}

	material := strings.TrimSpace(req.Material)
	if material != "" && !option.Has(MaterialOptions(), material) {
		return httpx.Validation(fmt.Sprintf("管材只能是：%s", option.Labels(MaterialOptions())))
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return httpx.Validation("管段编号不能为空")
	}

	target.Code = code
	target.Name = strings.TrimSpace(req.Name)
	target.District = strings.TrimSpace(req.District)
	target.RoadName = strings.TrimSpace(req.RoadName)
	target.PipeType = pipeType
	target.Material = material
	target.DiameterMm = req.DiameterMm
	target.LengthM = req.LengthM
	target.DepthM = req.DepthM
	target.StartManhole = strings.TrimSpace(req.StartManhole)
	target.EndManhole = strings.TrimSpace(req.EndManhole)
	target.BuildYear = req.BuildYear
	target.OwnerUnit = strings.TrimSpace(req.OwnerUnit)
	target.Remark = strings.TrimSpace(req.Remark)
	target.Status = status
	return nil
}

func notFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound("管段不存在")
	}
	return httpx.WrapInternal("查询管段失败", err)
}
