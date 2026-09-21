package cleaningtask_test

import (
	"context"
	"testing"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/testsupport"
)

func TestRecordEntryMovesTaskToInProgress(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "首次录入记录的任务")

	if task.Status != cleaningtask.StatusPending {
		t.Fatalf("新建任务应为待开工，实际 %s", task.Status)
	}

	fixture.CreateRecord(t, task.ID, 10)

	if got := fixture.Reload(t, task.ID).Status; got != cleaningtask.StatusInProgress {
		t.Fatalf("录入清淤记录后应自动进入清淤中，实际 %s", got)
	}
}

func TestCompleteRequiresCleaningRecord(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "未录入记录的任务")
	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Start(context.Background(), task.ID)))

	_, err := fixture.Tasks.Complete(context.Background(), task.ID)
	appErr := testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
	if appErr.Message == "" {
		t.Fatal("错误提示不应为空")
	}
}

func TestCompleteRejectsPendingTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "未开工的任务")

	_, err := fixture.Tasks.Complete(context.Background(), task.ID)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestCancelRequiresReason(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "待取消的任务")

	_, err := fixture.Tasks.Cancel(context.Background(), task.ID, "   ")
	testsupport.RequireAppError(t, err, httpx.CodeValidation)

	cancelled, err := fixture.Tasks.Cancel(context.Background(), task.ID, "汛期调度调整")
	testsupport.RequireNoError(t, err)
	if cancelled.Status != cleaningtask.StatusCancelled {
		t.Fatalf("期望任务状态为已取消，实际 %s", cancelled.Status)
	}
}

func TestCancelledTaskCannotBeStarted(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "已取消的任务")
	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Cancel(context.Background(), task.ID, "计划调整")))

	_, err := fixture.Tasks.Start(context.Background(), task.ID)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestAcceptedTaskCannotBeEdited(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "已验收的任务")
	_, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 92))
	testsupport.RequireNoError(t, err)

	_, err = fixture.Tasks.Update(context.Background(), task.ID, cleaningtask.SaveRequest{
		Title:         "修改后的标题",
		PipeSegmentID: fixture.Segment.ID,
		PlanStartDate: date.Today().AddDays(-3),
		PlanEndDate:   date.Today().AddDays(1),
	})
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestPlanEndBeforeStartIsRejected(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	_, err := fixture.Tasks.Create(context.Background(), cleaningtask.SaveRequest{
		Title:         "日期不合理的任务",
		PipeSegmentID: fixture.Segment.ID,
		PlanStartDate: date.Today(),
		PlanEndDate:   date.Today().AddDays(-5),
	})
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestTaskRequiresExistingSegment(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	_, err := fixture.Tasks.Create(context.Background(), cleaningtask.SaveRequest{
		Title:         "引用不存在管段的任务",
		PipeSegmentID: 99999,
		PlanStartDate: date.Today(),
		PlanEndDate:   date.Today().AddDays(2),
	})
	testsupport.RequireAppError(t, err, httpx.CodeNotFound)
}

func TestTaskCodeIsUniqueAndSequential(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	first := fixture.CreateTask(t, fixture.Segment.ID, "任务一")
	second := fixture.CreateTask(t, fixture.Segment.ID, "任务二")

	if first.Code == second.Code {
		t.Fatalf("任务编号重复: %s", first.Code)
	}
	if first.Code >= second.Code {
		t.Fatalf("同日任务编号应递增，实际 %s / %s", first.Code, second.Code)
	}
}

func TestDeleteTaskBlockedAfterRecord(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "已有记录的任务")
	fixture.CreateRecord(t, task.ID, 8)

	err := fixture.Tasks.Delete(context.Background(), task.ID)
	testsupport.RequireAppError(t, err, httpx.CodeConflict)
}

func TestAllowedActionsFollowStatus(t *testing.T) {
	cases := map[string][]string{
		cleaningtask.StatusPending:   {cleaningtask.ActionStart, cleaningtask.ActionEdit, cleaningtask.ActionCancel},
		cleaningtask.StatusCompleted: {cleaningtask.ActionAccept},
		cleaningtask.StatusAccepted:  {},
		cleaningtask.StatusCancelled: {},
	}
	for status, want := range cases {
		got := cleaningtask.AllowedActions(status)
		if len(got) != len(want) {
			t.Fatalf("状态 %s 期望可执行操作 %v，实际 %v", status, want, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("状态 %s 期望可执行操作 %v，实际 %v", status, want, got)
			}
		}
	}
}

// ignoreTask 丢弃任务返回值，只保留 error，便于在断言里直接使用。
func ignoreTask(_ *cleaningtask.CleaningTask, err error) error {
	return err
}
