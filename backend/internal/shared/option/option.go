// Package option 提供下拉选项（value/label）的统一表达方式。
//
// 后端把枚举的取值与中文名称集中维护：既用于接口校验时的提示文案，
// 也通过 /api/v1/meta/enums 下发给前端，避免前后端各写一份中文文案产生歧义。
package option

import "strings"

// Option 一个枚举选项。
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// List 用成对的 value/label 构造选项列表，例如 List("normal", "正常", "blocked", "已淤堵")。
func List(pairs ...string) []Option {
	if len(pairs)%2 != 0 {
		panic("option.List 需要成对的 value/label 参数")
	}
	out := make([]Option, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, Option{Value: pairs[i], Label: pairs[i+1]})
	}
	return out
}

// Has 判断 value 是否在选项中。
func Has(list []Option, value string) bool {
	for _, item := range list {
		if item.Value == value {
			return true
		}
	}
	return false
}

// Label 返回 value 对应的中文名称，找不到时原样返回。
func Label(list []Option, value string) string {
	for _, item := range list {
		if item.Value == value {
			return item.Label
		}
	}
	return value
}

// Labels 返回全部中文名称并用 / 连接，用于拼装校验报错提示。
func Labels(list []Option) string {
	names := make([]string, 0, len(list))
	for _, item := range list {
		names = append(names, item.Label)
	}
	return strings.Join(names, "/")
}

// Values 返回全部取值。
func Values(list []Option) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.Value)
	}
	return out
}
