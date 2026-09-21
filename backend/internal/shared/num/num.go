// Package num 提供数值处理的公共函数。
package num

import "math"

// 业务展示精度：清淤量、管长等指标统一保留 2 位小数。
const decimals = 2

// Round2 把浮点数四舍五入到 2 位小数。
//
// 浮点数累加会产生 33.599999999999994 这类尾差，接口返回前统一收敛，
// 避免前端展示与统计口径出现"看着不一致"的数字。
func Round2(value float64) float64 {
	factor := math.Pow(10, decimals)
	return math.Round(value*factor) / factor
}
