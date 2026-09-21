package pipesegment

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register 注册管段台账路由，并返回 service 供其他模块装配依赖。
func Register(router fiber.Router, db *gorm.DB) *Service {
	svc := NewService(NewRepository(db))
	handler := NewHandler(svc)

	group := router.Group("/pipe-segments")
	// 固定路径要注册在 /:id 之前，避免被参数路由抢先匹配。
	group.Get("/options", handler.Options)
	group.Get("", handler.List)
	group.Post("", handler.Create)
	group.Get("/:id", handler.Detail)
	group.Put("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
	group.Get("/:id/cleaning-history", handler.History)

	return svc
}
