package httpx

import "github.com/gofiber/fiber/v2"

// 分页参数的边界限制。
const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// PageQuery 分页查询参数。
type PageQuery struct {
	Page     int `query:"page"`
	PageSize int `query:"pageSize"`
}

// Normalize 把分页参数规整到合法范围。
func (p *PageQuery) Normalize() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Offset 计算 SQL 偏移量。
func (p *PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// ParsePage 从请求中读取分页参数并规整。
func ParsePage(c *fiber.Ctx) PageQuery {
	page := PageQuery{
		Page:     c.QueryInt("page", DefaultPage),
		PageSize: c.QueryInt("pageSize", DefaultPageSize),
	}
	page.Normalize()
	return page
}

// TrimmedQuery 读取并去除首尾空白的查询参数。
func TrimmedQuery(c *fiber.Ctx, key string) string {
	return trimSpace(c.Query(key))
}
