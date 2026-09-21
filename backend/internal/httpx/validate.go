package httpx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// validatorEngine 全局复用同一个校验器实例（内部会缓存反射信息，并发安全）。
var validatorEngine = newValidator()

func newValidator() *validator.Validate {
	instance := validator.New(validator.WithRequiredStructEnabled())
	// 用结构体标签 label 作为字段名，报错时直接输出中文名称。
	instance.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.TrimSpace(strings.SplitN(field.Tag.Get("label"), ",", 2)[0])
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	return instance
}

// BindAndValidate 解析请求体并做字段校验。
func BindAndValidate(c *fiber.Ctx, payload any) error {
	if err := c.BodyParser(payload); err != nil {
		return BadRequestf("请求体解析失败，请检查 JSON 格式：%v", err)
	}
	return Validate(payload)
}

// Validate 对结构体做字段校验，返回带中文提示的 AppError。
func Validate(payload any) error {
	if err := validatorEngine.Struct(payload); err != nil {
		var fieldErrors validator.ValidationErrors
		if errors.As(err, &fieldErrors) && len(fieldErrors) > 0 {
			return Validation(describe(fieldErrors[0]))
		}
		return Validation(err.Error())
	}
	return nil
}

// describe 把校验失败翻译成中文提示。
func describe(fieldErr validator.FieldError) string {
	field := fieldErr.Field()
	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("%s不能为空", field)
	case "min":
		return fmt.Sprintf("%s长度或取值不能小于 %s", field, fieldErr.Param())
	case "max":
		return fmt.Sprintf("%s长度或取值不能大于 %s", field, fieldErr.Param())
	case "gt":
		return fmt.Sprintf("%s必须大于 %s", field, fieldErr.Param())
	case "gte":
		return fmt.Sprintf("%s不能小于 %s", field, fieldErr.Param())
	case "lt":
		return fmt.Sprintf("%s必须小于 %s", field, fieldErr.Param())
	case "lte":
		return fmt.Sprintf("%s不能大于 %s", field, fieldErr.Param())
	case "len":
		return fmt.Sprintf("%s长度必须为 %s", field, fieldErr.Param())
	case "oneof":
		return fmt.Sprintf("%s只能是 %s 之一", field, strings.Join(strings.Fields(fieldErr.Param()), "/"))
	case "email":
		return fmt.Sprintf("%s不是合法的邮箱地址", field)
	case "url":
		return fmt.Sprintf("%s不是合法的链接", field)
	case "datetime":
		return fmt.Sprintf("%s日期格式不正确", field)
	default:
		return fmt.Sprintf("%s不符合要求（%s=%s）", field, fieldErr.Tag(), fieldErr.Param())
	}
}
