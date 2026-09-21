package dashboard

import (
	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
)

// Handler 看板 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Overview 总览指标。
func (h *Handler) Overview(c *fiber.Ctx) error {
	overview, err := h.svc.Overview(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, overview)
}

// DistrictStats 片区统计。
func (h *Handler) DistrictStats(c *fiber.Ctx) error {
	stats, err := h.svc.DistrictStats(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, stats)
}

// PendingAcceptance 待验收任务清单。
func (h *Handler) PendingAcceptance(c *fiber.Ctx) error {
	items, err := h.svc.PendingAcceptance(c.UserContext(), c.QueryInt("limit", 10))
	if err != nil {
		return err
	}
	return httpx.OK(c, items)
}

// RecentRecords 最近清淤记录。
func (h *Handler) RecentRecords(c *fiber.Ctx) error {
	items, err := h.svc.RecentRecords(c.UserContext(), c.QueryInt("limit", 10))
	if err != nil {
		return err
	}
	return httpx.OK(c, items)
}
