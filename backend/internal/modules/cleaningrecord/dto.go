package cleaningrecord

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/shared/refx"
)

// SaveRequest 新增清淤记录的请求体。
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

// UpdateRequest 修改清淤记录的请求体。
//
// 修改窗口的控制不依赖页面：即使绕过前端直接调用接口，后端也会按任务所处环节
// 与验收引用情况再次校验。修改人、修改原因必填，且与数值变更在同一事务内留痕。
type UpdateRequest struct {
	SaveRequest
	// ModifierName 修改人。
	ModifierName string `json:"modifierName" label:"修改人" validate:"required,max=32"`
	// ChangeReason 修改原因。
	ChangeReason string `json:"changeReason" label:"修改原因" validate:"required,max=255"`
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

// DetailResponse 记录详情：最新数值 + 修改窗口状态 + 版本履历概况。
type DetailResponse struct {
	Record *CleaningRecord `json:"record"`
	Task   *refx.TaskBrief `json:"task"`
	// Editable 当前是否仍处于允许修改的窗口。
	Editable bool `json:"editable"`
	// LockedReason 窗口关闭原因（已被验收引用 / 任务所处环节不允许），可编辑时为空。
	LockedReason string `json:"lockedReason"`
	// RevisionCount 历史版本数量（含当前版本）。
	RevisionCount int `json:"revisionCount"`
}

// RevisionItem 单个历史版本：版本元信息 + 该版本的完整业务快照。
type RevisionItem struct {
	Version      int             `json:"version"`
	Action       string          `json:"action"`
	ChangedBy    string          `json:"changedBy"`
	ChangeReason string          `json:"changeReason"`
	ChangedAt    string          `json:"changedAt"`
	Snapshot     VersionSnapshot `json:"snapshot"`
}

// RevisionListResponse 版本履历列表（最新版本在前）。
type RevisionListResponse struct {
	RecordID uint           `json:"recordId"`
	Code     string         `json:"code"`
	Items    []RevisionItem `json:"items"`
}
