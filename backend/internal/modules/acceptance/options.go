package acceptance

import "github.com/drainage/desilting/internal/shared/option"

// ResultOptions 验收结论选项。
func ResultOptions() []option.Option {
	return option.List(
		ResultPass, "合格",
		ResultRework, "需整改",
	)
}
