package cleaningrecord

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/refx"
)

// SaveRequest 新增或修改清淤记录的请求体。
type SaveRequest struct {
	TaskID             uint      `json:"taskId" label:"关联任务" validate:"required"`
	CleanedAt          date.Date `json:"cleanedAt" label:"清淤日期"`
	LengthM            float64   `json:"lengthM" label:"清淤长度(m)" validate:"gt=0,lte=100000"`
	SludgeVolumeM3     float64   `json:"sludgeVolumeM3" label:"清淤量(m³)" validate:"gt=0,lte=100000"`
	WaterVolumeM3      float64   `json:"waterVolumeM3" label:"用水量(m³)" validate:"gte=0,lte=100000"`
	PersonnelCount     int       `json:"personnelCount" label:"作业人数" validate:"gt=0,lte=500"`
	Method             string    `json:"method" label:"清淤方式"`
	Equipment          string    `json:"equipment" label:"主要设备" validate:"max=128"`
	Weather            string    `json:"weather" label:"天气"`
	SludgeDisposalSite string    `json:"sludgeDisposalSite" label:"污泥消纳点" validate:"max=128"`
	SafetyMeasures     string    `json:"safetyMeasures" label:"安全措施" validate:"max=1000"`
	ProblemFound       string    `json:"problemFound" label:"发现的问题" validate:"max=1000"`
	RecorderName       string    `json:"recorderName" label:"记录人" validate:"required,max=32"`
	Remark             string    `json:"remark" label:"备注" validate:"max=1000"`
}

// ListQuery 清淤记录列表查询条件。
type ListQuery struct {
	Keyword   string
	TaskID    uint
	SegmentID uint
	Method    string
	Weather   string
	DateFrom  *date.Date
	DateTo    *date.Date
	Page      httpx.PageQuery
}

// ParseListQuery 解析列表查询条件。
func ParseListQuery(c *fiber.Ctx) (ListQuery, error) {
	query := ListQuery{
		Keyword:   httpx.TrimmedQuery(c, "keyword"),
		TaskID:    uint(c.QueryInt("taskId", 0)),
		SegmentID: uint(c.QueryInt("segmentId", 0)),
		Method:    httpx.TrimmedQuery(c, "method"),
		Weather:   httpx.TrimmedQuery(c, "weather"),
		Page:      httpx.ParsePage(c),
	}
	from, err := parseDateParam(c, "dateFrom", "清淤日期起")
	if err != nil {
		return ListQuery{}, err
	}
	to, err := parseDateParam(c, "dateTo", "清淤日期止")
	if err != nil {
		return ListQuery{}, err
	}
	query.DateFrom = from
	query.DateTo = to
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

// ListItem 记录列表项：记录本体 + 所属任务与管段信息。
type ListItem struct {
	CleaningRecord
	Task *refx.TaskBrief `json:"task"`
}

// DetailResponse 记录详情。
type DetailResponse struct {
	Record *CleaningRecord `json:"record"`
	Task   *refx.TaskBrief `json:"task"`
}
