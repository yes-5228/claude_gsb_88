// Package meta 提供系统级元数据（枚举字典），作为前后端共享的唯一文案来源。
package meta

import (
	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/acceptance"
	"github.com/drainage/desilting/internal/modules/cleaningrecord"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/shared/option"
)

// Enums 全部枚举字典。
type Enums struct {
	PipeTypes         []option.Option `json:"pipeTypes"`
	SegmentStatuses   []option.Option `json:"segmentStatuses"`
	Materials         []option.Option `json:"materials"`
	TaskStatuses      []option.Option `json:"taskStatuses"`
	TaskPriorities    []option.Option `json:"taskPriorities"`
	TaskSources       []option.Option `json:"taskSources"`
	CleaningMethods   []option.Option `json:"cleaningMethods"`
	Weathers          []option.Option `json:"weathers"`
	AcceptanceResults []option.Option `json:"acceptanceResults"`
}

// Register 注册元数据路由。
func Register(router fiber.Router) {
	group := router.Group("/meta")
	group.Get("/enums", func(c *fiber.Ctx) error {
		return httpx.OK(c, Enums{
			PipeTypes:         pipesegment.PipeTypeOptions(),
			SegmentStatuses:   pipesegment.StatusOptions(),
			Materials:         pipesegment.MaterialOptions(),
			TaskStatuses:      cleaningtask.StatusOptions(),
			TaskPriorities:    cleaningtask.PriorityOptions(),
			TaskSources:       cleaningtask.SourceOptions(),
			CleaningMethods:   cleaningtask.MethodOptions(),
			Weathers:          cleaningrecord.WeatherOptions(),
			AcceptanceResults: acceptance.ResultOptions(),
		})
	})
}
