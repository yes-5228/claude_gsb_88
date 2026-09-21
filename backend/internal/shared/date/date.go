// Package date 提供只保留年月日的日期类型，用于清淤计划、验收等业务日期字段。
//
// 统一约定：内部一律以 UTC 零点保存。这样无论 JSON 序列化还是数据库读写，
// 都不会因为时区换算出现日期跨天的问题（对外的 JSON 始终是 YYYY-MM-DD）。
package date

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Layout 是对外约定的日期格式。
const Layout = "2006-01-02"

// fallbackLayouts 兼容数据库驱动可能返回的几种时间文本格式。
var fallbackLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05",
}

// Date 仅包含年月日的日期，空值用零值表示。
type Date struct {
	time.Time
}

// New 由 time.Time 构造日期，丢弃时分秒并统一到 UTC 零点。
func New(t time.Time) Date {
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

// Parse 解析 YYYY-MM-DD 或 RFC3339 文本。
func Parse(value string) (Date, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Date{}, fmt.Errorf("日期不能为空")
	}
	if t, err := time.ParseInLocation(Layout, trimmed, time.UTC); err == nil {
		return Date{t}, nil
	}
	for _, layout := range fallbackLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return New(t), nil
		}
	}
	return Date{}, fmt.Errorf("日期格式不正确，应为 %s，实际为 %q", Layout, trimmed)
}

// MustParse 解析失败时 panic，仅用于种子数据与测试。
func MustParse(value string) Date {
	parsed, err := Parse(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

// Ptr 返回日期指针，便于构造可选字段。
func Ptr(value string) *Date {
	parsed := MustParse(value)
	return &parsed
}

// Today 返回当前日期。
func Today() Date {
	return New(time.Now())
}

// AddDays 返回偏移若干天后的日期。
func (d Date) AddDays(days int) Date {
	return New(d.Time.AddDate(0, 0, days))
}

func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Time.Format(Layout)
}

func (d Date) IsZero() bool {
	return d.Time.IsZero()
}

// Before 判断是否早于 other。
func (d Date) Before(other Date) bool {
	return d.Time.Before(other.Time)
}

// After 判断是否晚于 other。
func (d Date) After(other Date) bool {
	return d.Time.After(other.Time)
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*d = Date{}
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("日期格式不正确，应为 %s 文本", Layout)
	}
	if strings.TrimSpace(raw) == "" {
		*d = Date{}
		return nil
	}
	parsed, err := Parse(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// GormDataType 让 SQLite 与 PostgreSQL 都建为 date 列。
func (Date) GormDataType() string {
	return "date"
}

// Value 实现 driver.Valuer，零值写入 NULL。
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.Time.UTC(), nil
}

// Scan 实现 sql.Scanner：按取值自身的年月日解析，避免时区换算导致跨天。
func (d *Date) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*d = Date{}
	case time.Time:
		*d = New(v)
	case string:
		parsed, err := Parse(v)
		if err != nil {
			return err
		}
		*d = parsed
	case []byte:
		parsed, err := Parse(string(v))
		if err != nil {
			return err
		}
		*d = parsed
	default:
		return fmt.Errorf("无法解析日期字段：%v", value)
	}
	return nil
}
