package pipesegment

import "github.com/drainage/desilting/internal/shared/option"

// PipeTypeOptions 管段类型选项。
func PipeTypeOptions() []option.Option {
	return option.List(
		TypeRainwater, "雨水",
		TypeSewage, "污水",
		TypeCombined, "合流",
	)
}

// StatusOptions 管段运行状态选项。
func StatusOptions() []option.Option {
	return option.List(
		StatusNormal, "正常",
		StatusAttention, "需关注",
		StatusBlocked, "已淤堵",
	)
}

// MaterialOptions 常用管材选项。
func MaterialOptions() []option.Option {
	return option.List(
		"concrete", "钢筋混凝土",
		"hdpe", "HDPE 双壁波纹管",
		"ductile_iron", "球墨铸铁",
		"pvc", "PVC-U",
		"grp", "玻璃钢夹砂",
		"other", "其他",
	)
}
