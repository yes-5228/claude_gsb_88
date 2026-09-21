package httpx

import "strings"

// trimSpace 是 strings.TrimSpace 的简写，便于在参数解析里连续调用。
func trimSpace(value string) string {
	return strings.TrimSpace(value)
}
