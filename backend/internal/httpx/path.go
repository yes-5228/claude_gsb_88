package httpx

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// PathID 解析路径参数中的自增主键。
//
// label 用于拼装报错文案，例如 PathID(c, "id", "管段") 会返回"管段 ID 不合法"。
func PathID(c *fiber.Ctx, name, label string) (uint, error) {
	raw := c.Params(name)
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		return 0, BadRequest(fmt.Sprintf("%s ID 不合法：%s", label, raw))
	}
	return uint(parsed), nil
}
