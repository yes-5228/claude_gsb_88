package pipesegment

import (
	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
)

// Handler 管段台账 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 管段列表。
func (h *Handler) List(c *fiber.Ctx) error {
	query := ParseListQuery(c)
	items, total, err := h.svc.List(c.UserContext(), query)
	if err != nil {
		return err
	}
	return httpx.OKPage(c, items, total, query.Page.Page, query.Page.PageSize)
}

// Create 新增管段。
func (h *Handler) Create(c *fiber.Ctx) error {
	var req SaveRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	segment, err := h.svc.Create(c.UserContext(), req)
	if err != nil {
		return err
	}
	return httpx.Created(c, segment)
}

// Detail 管段详情。
func (h *Handler) Detail(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "管段")
	if err != nil {
		return err
	}
	detail, err := h.svc.Detail(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, detail)
}

// Update 修改管段。
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "管段")
	if err != nil {
		return err
	}
	var req SaveRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	segment, err := h.svc.Update(c.UserContext(), id, req)
	if err != nil {
		return err
	}
	return httpx.Message(c, "管段信息已更新", segment)
}

// Delete 删除管段。
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "管段")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return err
	}
	return httpx.Message(c, "管段已删除", fiber.Map{"id": id})
}

// History 管段清淤履历。
func (h *Handler) History(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "管段")
	if err != nil {
		return err
	}
	items, err := h.svc.History(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, items)
}

// Options 下拉选项。
func (h *Handler) Options(c *fiber.Ctx) error {
	data, err := h.svc.Options(c.UserContext(), httpx.TrimmedQuery(c, "keyword"))
	if err != nil {
		return err
	}
	return httpx.OK(c, data)
}
