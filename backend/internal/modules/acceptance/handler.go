package acceptance

import (
	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
)

// Handler 验收记录 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 验收记录列表。
func (h *Handler) List(c *fiber.Ctx) error {
	query, err := ParseListQuery(c)
	if err != nil {
		return err
	}
	items, total, err := h.svc.List(c.UserContext(), query)
	if err != nil {
		return err
	}
	return httpx.OKPage(c, items, total, query.Page.Page, query.Page.PageSize)
}

// Create 登记验收。
func (h *Handler) Create(c *fiber.Ctx) error {
	var req SaveRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	record, err := h.svc.Create(c.UserContext(), req)
	if err != nil {
		return err
	}
	return httpx.Created(c, record)
}

// Detail 验收详情。
func (h *Handler) Detail(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "验收记录")
	if err != nil {
		return err
	}
	detail, err := h.svc.Detail(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, detail)
}

// Rectify 登记整改完成。
func (h *Handler) Rectify(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "验收记录")
	if err != nil {
		return err
	}
	var req RectifyRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	record, err := h.svc.Rectify(c.UserContext(), id, req)
	if err != nil {
		return err
	}
	return httpx.Message(c, "整改完成已登记，可重新提交完工报验", record)
}

// Delete 删除验收记录。
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "验收记录")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return err
	}
	return httpx.Message(c, "验收记录已删除", fiber.Map{"id": id})
}
