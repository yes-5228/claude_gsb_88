package cleaningrecord

import (
	"fmt"
	"time"

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

// UpdateRequest 修改清淤记录的请求体。
//
// 相比录入，修改必须额外说明修改人与修改原因，用于生成修改留痕；
// 两者缺一不可，保证"数值变了就一定有对应的留痕"。
type UpdateRequest struct {
	SaveRequest
	EditorName   string `json:"editorName" label:"修改人" validate:"required,max=32"`
	ChangeReason string `json:"changeReason" label:"修改原因" validate:"required,max=200"`
}

// EditWindow 清淤记录的修改窗口状态。
//
// 窗口起止由任务所处环节决定：任务处于「待开工」「清淤中」时窗口开放，
// 完工报验后关闭；被验收记录引用的记录只能查看历史版本。
type EditWindow struct {
	Open   bool   `json:"open"`
	Reason string `json:"reason"`
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
	Record         *CleaningRecord `json:"record"`
	Task           *refx.TaskBrief `json:"task"`
	EditWindow     EditWindow      `json:"editWindow"`
	CurrentVersion int             `json:"currentVersion"`
}

// RecordVersion 记录的一个历史版本：留痕快照或当前版本（current = true）。
//
// 留痕版本上的修改人 / 修改原因 / 修改时间描述的是"该版本被下一版本取代"的那次调整；
// 当前版本没有这些字段。
type RecordVersion struct {
	Version            int        `json:"version"`
	Current            bool       `json:"current"`
	EditorName         string     `json:"editorName"`
	ChangeReason       string     `json:"changeReason"`
	ChangedAt          *time.Time `json:"changedAt"`
	CleanedAt          date.Date  `json:"cleanedAt"`
	LengthM            float64    `json:"lengthM"`
	SludgeVolumeM3     float64    `json:"sludgeVolumeM3"`
	WaterVolumeM3      float64    `json:"waterVolumeM3"`
	PersonnelCount     int        `json:"personnelCount"`
	Method             string     `json:"method"`
	Equipment          string     `json:"equipment"`
	Weather            string     `json:"weather"`
	SludgeDisposalSite string     `json:"sludgeDisposalSite"`
	SafetyMeasures     string     `json:"safetyMeasures"`
	ProblemFound       string     `json:"problemFound"`
	RecorderName       string     `json:"recorderName"`
	Remark             string     `json:"remark"`
}

// VersionListResponse 版本列表：versions 按版本号升序，最后一项为当前版本。
type VersionListResponse struct {
	CurrentVersion int             `json:"currentVersion"`
	Versions       []RecordVersion `json:"versions"`
}
