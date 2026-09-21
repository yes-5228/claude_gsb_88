package cleaningtask

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/refx"
)

// SaveRequest 新增或修改清淤任务的请求体。
type SaveRequest struct {
	Title         string    `json:"title" label:"任务标题" validate:"required,max=128"`
	PipeSegmentID uint      `json:"pipeSegmentId" label:"关联管段" validate:"required"`
	Priority      string    `json:"priority" label:"优先级"`
	Source        string    `json:"source" label:"任务来源"`
	Method        string    `json:"method" label:"清淤方式"`
	PlanStartDate date.Date `json:"planStartDate" label:"计划开始日期"`
	PlanEndDate   date.Date `json:"planEndDate" label:"计划完成日期"`
	TeamName      string    `json:"teamName" label:"实施班组" validate:"max=64"`
	LeaderName    string    `json:"leaderName" label:"现场负责人" validate:"max=32"`
	LeaderPhone   string    `json:"leaderPhone" label:"联系电话" validate:"max=32"`
	Description   string    `json:"description" label:"任务说明" validate:"max=1000"`
}

// CancelRequest 取消任务请求体。
type CancelRequest struct {
	Reason string `json:"reason" label:"取消原因" validate:"required,max=255"`
}

// ListQuery 任务列表查询条件。
type ListQuery struct {
	Keyword       string
	Status        string
	District      string
	Priority      string
	Source        string
	PipeSegmentID uint
	PlanFrom      *date.Date
	PlanTo        *date.Date
	Page          httpx.PageQuery
}

// ParseListQuery 解析任务列表查询条件。
func ParseListQuery(c *fiber.Ctx) (ListQuery, error) {
	query := ListQuery{
		Keyword:       httpx.TrimmedQuery(c, "keyword"),
		Status:        httpx.TrimmedQuery(c, "status"),
		District:      httpx.TrimmedQuery(c, "district"),
		Priority:      httpx.TrimmedQuery(c, "priority"),
		Source:        httpx.TrimmedQuery(c, "source"),
		PipeSegmentID: uint(c.QueryInt("pipeSegmentId", 0)),
		Page:          httpx.ParsePage(c),
	}
	from, err := parseDateParam(c, "planFrom", "计划开始日期起")
	if err != nil {
		return ListQuery{}, err
	}
	to, err := parseDateParam(c, "planTo", "计划开始日期止")
	if err != nil {
		return ListQuery{}, err
	}
	query.PlanFrom = from
	query.PlanTo = to
	return query, nil
}

func parseDateParam(c *fiber.Ctx, key, label string) (*date.Date, error) {
	raw := httpx.TrimmedQuery(c, key)
	if raw == "" {
		return nil, nil
	}
	parsed, err := date.Parse(raw)
	if err != nil {
		return nil, httpx.BadRequest(fmt.Sprintf("%s格式不正确，应为 YYYY-MM-DD", label))
	}
	return &parsed, nil
}

// ListItem 任务列表项：任务本体 + 管段信息 + 清淤汇总。
type ListItem struct {
	CleaningTask
	Segment      *pipesegment.Brief `json:"segment"`
	RecordTotals refx.RecordTotals  `json:"recordTotals"`
}

// DetailResponse 任务详情：任务 + 管段 + 清淤汇总 + 验收结论 + 可执行操作。
type DetailResponse struct {
	Task           *CleaningTask         `json:"task"`
	Segment        *pipesegment.Brief    `json:"segment"`
	RecordTotals   refx.RecordTotals     `json:"recordTotals"`
	Acceptance     *refx.AcceptanceBrief `json:"acceptance"`
	AllowedActions []string              `json:"allowedActions"`
}
