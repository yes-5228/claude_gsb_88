// Package middleware 提供请求 ID、访问日志、异常兜底等通用中间件。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/drainage/desilting/internal/httpx"
)

// RequestIDHeader 请求 ID 的头部名称，便于链路追踪。
const RequestIDHeader = "X-Request-ID"

const requestIDKey = "requestID"

// RequestID 为每个请求生成（或透传）请求 ID。
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(RequestIDHeader, id)
		c.Locals(requestIDKey, id)
		return c.Next()
	}
}

// AccessLog 记录访问日志，包含耗时与状态码。
func AccessLog(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			// 错误真正的响应状态码由 ErrorHandler 决定，这里取解析结果才是日志里的真实值。
			status = httpx.StatusOf(err)
		}
		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"durationMs", time.Since(start).Milliseconds(),
			"requestId", RequestIDOf(c),
			"ip", c.IP(),
		}
		if err != nil {
			// 4xx 属于客户端问题，降级为 WARN，避免干扰真实故障排查。
			if status >= fiber.StatusInternalServerError {
				log.Error("请求处理失败", append(attrs, "error", err.Error())...)
			} else {
				log.Warn("请求被拒绝", append(attrs, "error", err.Error())...)
			}
			return err
		}
		log.Info("请求完成", attrs...)
		return nil
	}
}

// Recover 兜底 panic，保证接口始终返回统一信封而不是连接中断。
func Recover(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("捕获到未处理的 panic",
					"panic", fmt.Sprint(recovered),
					"path", c.Path(),
					"requestId", RequestIDOf(c),
					"stack", string(debug.Stack()),
				)
				_ = httpx.WriteError(c, httpx.Internal("服务器内部错误，请稍后重试"))
			}
		}()
		return c.Next()
	}
}

// CORS 允许前端在本地开发时直接跨域访问后端（生产环境由 nginx 反向代理，同源）。
func CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,"+RequestIDHeader)
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

// RequestIDOf 读取当前请求的请求 ID。
func RequestIDOf(c *fiber.Ctx) string {
	if id, ok := c.Locals(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
