package cleaningtask

import (
	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
)

// Handler 清淤任务 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 任务列表。
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

// Create 登记任务。
func (h *Handler) Create(c *fiber.Ctx) error {
	var req SaveRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	task, err := h.svc.Create(c.UserContext(), req)
	if err != nil {
		return err
	}
	return httpx.Created(c, task)
}

// Detail 任务详情。
func (h *Handler) Detail(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "任务")
	if err != nil {
		return err
	}
	detail, err := h.svc.Detail(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, detail)
}

// Update 修改任务。
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "任务")
	if err != nil {
		return err
	}
	var req SaveRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	task, err := h.svc.Update(c.UserContext(), id, req)
	if err != nil {
		return err
	}
	return httpx.Message(c, "任务已更新", task)
}

// Delete 删除任务。
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "任务")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return err
	}
	return httpx.Message(c, "任务已删除", fiber.Map{"id": id})
}

// Start 开工。
func (h *Handler) Start(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "任务")
	if err != nil {
		return err
	}
	task, err := h.svc.Start(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.Message(c, "任务已开工", task)
}

// Complete 完工报验。
func (h *Handler) Complete(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "任务")
	if err != nil {
		return err
	}
	task, err := h.svc.Complete(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.Message(c, "任务已提交完工报验，等待验收", task)
}

// Cancel 取消任务。
func (h *Handler) Cancel(c *fiber.Ctx) error {
	id, err := httpx.PathID(c, "id", "任务")
	if err != nil {
		return err
	}
	var req CancelRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	task, err := h.svc.Cancel(c.UserContext(), id, req.Reason)
	if err != nil {
		return err
	}
	return httpx.Message(c, "任务已取消", task)
}
