package acceptance_test

import (
	"context"
	"testing"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/acceptance"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/testsupport"
)

func TestAcceptanceRequiresCompletedTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "尚未完工的任务")

	_, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestAcceptanceRequiresCleaningRecord(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "没有记录的任务")
	// 直接改状态模拟"没有清淤记录却处于待验收"的异常数据
	testsupport.RequireNoError(t, fixture.DB.Model(&cleaningtask.CleaningTask{}).
		Where("id = ?", task.ID).Update("status", cleaningtask.StatusCompleted).Error)

	_, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))
	appErr := testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
	if appErr.Message == "" {
		t.Fatal("错误提示不应为空")
	}
}

func TestPassAcceptsTaskAndUpdatesSegmentStats(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "验收合格的任务")

	record, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 92))
	testsupport.RequireNoError(t, err)
	if record.Code == "" {
		t.Fatal("验收编号不应为空")
	}

	updated := fixture.Reload(t, task.ID)
	if updated.Status != cleaningtask.StatusAccepted {
		t.Fatalf("验收合格后任务状态应为已验收，实际 %s", updated.Status)
	}
	if updated.AcceptedAt == nil {
		t.Fatal("验收合格后应记录验收时间")
	}

	segment, err := fixture.Segments.FindByID(context.Background(), fixture.Segment.ID)
	testsupport.RequireNoError(t, err)
	if segment.CleanedTimes != 1 {
		t.Fatalf("期望管段清淤次数为 1，实际 %d", segment.CleanedTimes)
	}
	if segment.LastCleanedAt == nil || segment.LastCleanedAt.String() != date.Today().AddDays(-1).String() {
		t.Fatalf("期望最近清淤日期取清淤记录日期，实际 %+v", segment.LastCleanedAt)
	}
}

func TestReworkFlowReturnsTaskForRectification(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "验收需整改的任务")

	rework, err := fixture.Acceptances.Create(context.Background(), testsupport.ReworkRequest(task.ID))
	testsupport.RequireNoError(t, err)
	if rework.RectifiedAt != nil {
		t.Fatal("新登记的整改记录不应带有整改完成时间")
	}

	reopened := fixture.Reload(t, task.ID)
	if reopened.Status != cleaningtask.StatusInProgress {
		t.Fatalf("验收需整改应把任务退回清淤中，实际 %s", reopened.Status)
	}

	// 未登记整改完成前不允许再次验收
	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Complete(context.Background(), task.ID)))
	_, err = fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))
	appErr := testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
	if appErr.Message == "" {
		t.Fatal("错误提示不应为空")
	}

	// 登录整改完成 -> 重新报验 -> 验收合格
	_, err = fixture.Acceptances.Rectify(context.Background(), rework.ID, acceptance.RectifyRequest{
		RectifiedAt:   date.Today(),
		Rectification: "已联系维修班组完成错口修复并重新清淤",
	})
	testsupport.RequireNoError(t, err)

	// 整改登记完成时任务已经回到待验收状态，可以直接复验
	_, err = fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 88))
	testsupport.RequireNoError(t, err)

	if got := fixture.Reload(t, task.ID).Status; got != cleaningtask.StatusAccepted {
		t.Fatalf("复验合格后任务状态应为已验收，实际 %s", got)
	}
}

func TestReworkRequiresIssuesAndDeadline(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "整改信息缺失的任务")

	request := testsupport.ReworkRequest(task.ID)
	request.Issues = "  "
	_, err := fixture.Acceptances.Create(context.Background(), request)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)

	request = testsupport.ReworkRequest(task.ID)
	request.RectifyDeadline = nil
	_, err = fixture.Acceptances.Create(context.Background(), request)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestPassWithLowScoreIsRejected(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "评分矛盾的任务")

	_, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 45))
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestDuplicatePassIsRejected(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "重复验收的任务")
	testsupport.RequireNoError(t, ignoreAcceptance(fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))))

	_, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 95))
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestPassedAcceptanceCannotBeDeleted(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "删除校验的任务")
	record, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))
	testsupport.RequireNoError(t, err)

	err = fixture.Acceptances.Delete(context.Background(), record.ID)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestRectifyRejectsPassedAcceptance(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "合格记录的任务")
	record, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))
	testsupport.RequireNoError(t, err)

	_, err = fixture.Acceptances.Rectify(context.Background(), record.ID, acceptance.RectifyRequest{
		Rectification: "无需整改",
	})
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestListAcceptancesFiltersPendingRectify(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	reworkTask := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "待整改的任务")
	testsupport.RequireNoError(t, ignoreAcceptance(fixture.Acceptances.Create(context.Background(), testsupport.ReworkRequest(reworkTask.ID))))

	passTask := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "合格的任务")
	testsupport.RequireNoError(t, ignoreAcceptance(fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(passTask.ID, 90))))

	items, total, err := fixture.Acceptances.List(context.Background(), acceptance.ListQuery{
		PendingRectify: true,
		Page:           httpx.PageQuery{Page: 1, PageSize: 10},
	})
	testsupport.RequireNoError(t, err)
	if total != 1 || len(items) != 1 {
		t.Fatalf("期望筛出 1 条待整改记录，实际 total=%d len=%d", total, len(items))
	}
	if items[0].Task == nil || items[0].Task.ID != reworkTask.ID {
		t.Fatalf("列表项应带出所属任务信息，实际 %+v", items[0].Task)
	}
}

func TestSegmentStatusStaysNormalAfterPass(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	testsupport.RequireNoError(t, fixture.DB.Model(&pipesegment.PipeSegment{}).
		Where("id = ?", fixture.Segment.ID).Update("status", pipesegment.StatusBlocked).Error)

	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "淤堵管段清淤任务")
	testsupport.RequireNoError(t, ignoreAcceptance(fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))))

	segment, err := fixture.Segments.FindByID(context.Background(), fixture.Segment.ID)
	testsupport.RequireNoError(t, err)
	if segment.Status != pipesegment.StatusNormal {
		t.Fatalf("验收合格后管段状态应恢复为正常，实际 %s", segment.Status)
	}
}

func ignoreTask(_ *cleaningtask.CleaningTask, err error) error {
	return err
}

func ignoreAcceptance(_ *acceptance.AcceptanceRecord, err error) error {
	return err
}
