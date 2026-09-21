// Package httpx 统一接口的响应结构、错误码与参数校验。
//
// 所有接口都返回同一层信封结构，前端只需解析一次：
//
//	{ "code": 0, "message": "操作成功", "data": {...}, "timestamp": 1710000000000 }
//
// code 为 0 表示成功，非 0 为业务错误码（与 HTTP 状态码配合使用）。
package httpx

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Envelope 统一响应信封。
type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// PageData 分页数据结构。
type PageData struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func timestamp() int64 {
	return time.Now().UnixMilli()
}

func respond(c *fiber.Ctx, status, code int, message string, data any) error {
	// 明确带上 charset：部分 HTTP 客户端在没有 charset 时会按本地编码解析中文。
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return c.Status(status).JSON(Envelope{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: timestamp(),
	})
}

// OK 返回 200 成功响应。
func OK(c *fiber.Ctx, data any) error {
	return respond(c, fiber.StatusOK, CodeOK, "操作成功", data)
}

// Message 返回 200 成功响应并自定义提示文案。
func Message(c *fiber.Ctx, message string, data any) error {
	return respond(c, fiber.StatusOK, CodeOK, message, data)
}

// Created 返回 201 新建成功响应。
func Created(c *fiber.Ctx, data any) error {
	return respond(c, fiber.StatusCreated, CodeOK, "创建成功", data)
}

// OKPage 返回分页结果。
func OKPage(c *fiber.Ctx, list any, total int64, page, pageSize int) error {
	return OK(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}
