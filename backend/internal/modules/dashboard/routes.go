package dashboard

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register 注册看板路由。
func Register(router fiber.Router, db *gorm.DB) *Service {
	svc := NewService(db)
	handler := NewHandler(svc)

	group := router.Group("/dashboard")
	group.Get("/overview", handler.Overview)
	group.Get("/district-stats", handler.DistrictStats)
	group.Get("/pending-acceptance", handler.PendingAcceptance)
	group.Get("/recent-records", handler.RecentRecords)

	return svc
}
