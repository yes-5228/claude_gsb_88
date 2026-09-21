package httpx

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// 业务错误码。前两位对应 HTTP 状态码，便于前端按区间兜底处理。
const (
	CodeOK = 0

	CodeBadRequest   = 40000 // 请求参数错误
	CodeValidation   = 40001 // 字段校验不通过
	CodeInvalidState = 40002 // 当前业务状态不允许该操作
	CodeNotFound     = 40400 // 资源不存在
	CodeConflict     = 40900 // 唯一性或业务冲突
	CodeInternal     = 50000 // 服务内部错误
)

// AppError 业务错误，携带 HTTP 状态码与业务错误码。
type AppError struct {
	Status  int
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 支持 errors.Is / errors.As 透传底层错误。
func (e *AppError) Unwrap() error {
	return e.Cause
}

// BadRequest 参数错误。
func BadRequest(message string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Code: CodeBadRequest, Message: message}
}

// BadRequestf 参数错误（格式化）。
func BadRequestf(format string, args ...any) *AppError {
	return BadRequest(fmt.Sprintf(format, args...))
}

// Validation 字段校验不通过。
func Validation(message string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Code: CodeValidation, Message: message}
}

// InvalidState 当前业务状态不允许该操作。
func InvalidState(message string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Code: CodeInvalidState, Message: message}
}

// NotFound 资源不存在。
func NotFound(message string) *AppError {
	return &AppError{Status: fiber.StatusNotFound, Code: CodeNotFound, Message: message}
}

// Conflict 唯一性或业务冲突。
func Conflict(message string) *AppError {
	return &AppError{Status: fiber.StatusConflict, Code: CodeConflict, Message: message}
}

// Internal 服务内部错误。
func Internal(message string) *AppError {
	return &AppError{Status: fiber.StatusInternalServerError, Code: CodeInternal, Message: message}
}

// WrapInternal 把底层错误包装为服务内部错误，保留原因便于日志排查。
func WrapInternal(message string, cause error) *AppError {
	return &AppError{Status: fiber.StatusInternalServerError, Code: CodeInternal, Message: message, Cause: cause}
}

// resolve 把任意错误解析为最终的 HTTP 状态码、业务错误码与提示文案。
func resolve(err error) (int, int, string) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status, appErr.Code, appErr.Message
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		message := fiberErr.Message
		switch fiberErr.Code {
		case fiber.StatusNotFound:
			message = "接口不存在"
		case fiber.StatusMethodNotAllowed:
			message = "请求方法不被支持"
		case fiber.StatusRequestEntityTooLarge:
			message = "请求内容过大"
		}
		return fiberErr.Code, fiberErr.Code * 100, message
	}

	return fiber.StatusInternalServerError, CodeInternal, "服务器内部错误，请稍后重试"
}

// WriteError 把任意错误渲染为统一信封，供 Fiber 的 ErrorHandler 使用。
func WriteError(c *fiber.Ctx, err error) error {
	status, code, message := resolve(err)
	return respond(c, status, code, message, nil)
}

// StatusOf 返回错误最终对应的 HTTP 状态码。
//
// 访问日志中间件在 c.Next() 返回后、ErrorHandler 执行前运行，
// 此时响应状态码还没有被写入，需要借助该函数得到真实状态码。
func StatusOf(err error) int {
	status, _, _ := resolve(err)
	return status
}
