package cleaningtask

import (
	"fmt"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/shared/option"
)

// 清淤任务状态。
const (
	StatusPending    = "pending"     // 待开工
	StatusInProgress = "in_progress" // 清淤中
	StatusCompleted  = "completed"   // 待验收
	StatusAccepted   = "accepted"    // 已验收
	StatusCancelled  = "cancelled"   // 已取消
)

// 前端可触发的操作标识。
const (
	ActionStart    = "start"
	ActionComplete = "complete"
	ActionAccept   = "accept"
	ActionCancel   = "cancel"
	ActionEdit     = "edit"
)

// allowedTransitions 任务状态的合法流转。
//
//	pending  --开工-->      in_progress
//	pending  --取消-->      cancelled
//	in_progress --完工报验--> completed
//	in_progress --取消-->   cancelled
//	completed --验收合格-->  accepted
//	completed --验收需整改--> in_progress
var allowedTransitions = map[string]map[string]bool{
	StatusPending: {
		StatusInProgress: true,
		StatusCancelled:  true,
	},
	StatusInProgress: {
		StatusCompleted: true,
		StatusCancelled: true,
	},
	StatusCompleted: {
		StatusAccepted:   true,
		StatusInProgress: true,
	},
	StatusAccepted:  {},
	StatusCancelled: {},
}

// StatusOptions 任务状态选项。
func StatusOptions() []option.Option {
	return option.List(
		StatusPending, "待开工",
		StatusInProgress, "清淤中",
		StatusCompleted, "待验收",
		StatusAccepted, "已验收",
		StatusCancelled, "已取消",
	)
}

// PriorityOptions 优先级选项。
func PriorityOptions() []option.Option {
	return option.List(
		PriorityLow, "低",
		PriorityNormal, "普通",
		PriorityHigh, "高",
		PriorityUrgent, "紧急",
	)
}

// SourceOptions 任务来源选项。
func SourceOptions() []option.Option {
	return option.List(
		SourcePlan, "年度计划",
		SourceInspection, "巡查发现",
		SourceComplaint, "投诉举报",
		SourceFlood, "汛期专项",
	)
}

// MethodOptions 清淤方式选项。
func MethodOptions() []option.Option {
	return option.List(
		MethodHighPressure, "高压水射流",
		MethodWinch, "绞车牵引",
		MethodGrab, "抓斗清掏",
		MethodManual, "人工清掏",
		MethodRobot, "管道机器人",
	)
}

// StatusLabel 返回状态中文名。
func StatusLabel(status string) string {
	return option.Label(StatusOptions(), status)
}

// CanTransition 判断状态流转是否合法。
func CanTransition(from, to string) bool {
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// TransitionError 生成状态流转失败的提示。
func TransitionError(from, to string) error {
	return httpx.InvalidState(fmt.Sprintf(
		"任务当前状态为「%s」，不允许变更为「%s」", StatusLabel(from), StatusLabel(to),
	))
}
