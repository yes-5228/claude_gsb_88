package cleaningrecord_test

import (
	"context"
	"testing"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/cleaningrecord"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/shared/date"
	"github.com/drainage/desilting/internal/testsupport"
)

func recordRequest(taskID uint) cleaningrecord.SaveRequest {
	return cleaningrecord.SaveRequest{
		TaskID:         taskID,
		CleanedAt:      date.Today().AddDays(-1),
		LengthM:        80,
		SludgeVolumeM3: 12.5,
		WaterVolumeM3:  40,
		PersonnelCount: 6,
		Method:         cleaningtask.MethodHighPressure,
		Weather:        cleaningrecord.WeatherSunny,
		RecorderName:   "李伟",
	}
}

// updateRequest 把录入请求包装成带修改人、修改原因的修改请求。
func updateRequest(req cleaningrecord.SaveRequest) cleaningrecord.UpdateRequest {
	return cleaningrecord.UpdateRequest{
		SaveRequest:  req,
		ModifierName: "王芳",
		ChangeReason: "现场复测后修正清淤量",
	}
}

func TestRecordRejectsFutureDate(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "日期校验任务")
	request := recordRequest(task.ID)
	request.CleanedAt = date.Today().AddDays(1)

	_, err := fixture.Records.Create(context.Background(), request)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestRecordRejectsUnknownWeather(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "天气校验任务")
	request := recordRequest(task.ID)
	request.Weather = "typhoon"

	_, err := fixture.Records.Create(context.Background(), request)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestRecordBlockedOnCompletedTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "待验收任务")

	_, err := fixture.Records.Create(context.Background(), recordRequest(task.ID))
	appErr := testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
	if appErr.Message == "" {
		t.Fatal("错误提示不应为空")
	}
}

func TestUpdateRecordBlockedWhenTaskCompleted(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "更新校验任务")
	record := fixture.CreateRecord(t, task.ID, 9)
	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Complete(context.Background(), task.ID)))

	request := recordRequest(task.ID)
	request.SludgeVolumeM3 = 20
	_, err := fixture.Records.Update(context.Background(), record.ID, updateRequest(request))
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestUpdateRecordRejectsChangingTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	first := fixture.CreateTask(t, fixture.Segment.ID, "任务一")
	second := fixture.CreateTask(t, fixture.Segment.ID, "任务二")
	record := fixture.CreateRecord(t, first.ID, 9)

	request := recordRequest(second.ID)
	_, err := fixture.Records.Update(context.Background(), record.ID, updateRequest(request))
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestUpdateRecordRequiresModifierAndReason(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "留痕必填任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	missingReason := updateRequest(recordRequest(task.ID))
	missingReason.ChangeReason = ""
	_, err := fixture.Records.Update(context.Background(), record.ID, missingReason)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)

	missingModifier := updateRequest(recordRequest(task.ID))
	missingModifier.ModifierName = "  "
	_, err = fixture.Records.Update(context.Background(), record.ID, missingModifier)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestUpdateRecordKeepsRevisionHistory(t *testing.T) {
	ctx := context.Background()
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "版本留痕任务")
	record := fixture.CreateRecord(t, task.ID, 10)

	// 录入后即有第 1 版。
	revisions, err := fixture.Records.Revisions(ctx, record.ID)
	testsupport.RequireNoError(t, err)
	if len(revisions.Items) != 1 || revisions.Items[0].Version != 1 {
		t.Fatalf("录入后期望 1 个版本，实际 %+v", revisions.Items)
	}
	if revisions.Items[0].Action != cleaningrecord.RevisionActionCreate {
		t.Fatalf("首版动作应为 create，实际 %s", revisions.Items[0].Action)
	}

	// 第 1 次修改：清淤量 10 -> 20。
	req := recordRequest(task.ID)
	req.SludgeVolumeM3 = 20
	updated, err := fixture.Records.Update(ctx, record.ID, updateRequest(req))
	testsupport.RequireNoError(t, err)
	if updated.Version != 2 {
		t.Fatalf("修改后版本号应为 2，实际 %d", updated.Version)
	}

	// 第 2 次修改：清淤长度 100 -> 130。
	req = recordRequest(task.ID)
	req.SludgeVolumeM3 = 20
	req.LengthM = 130
	second := updateRequest(req)
	second.ModifierName = "张强"
	second.ChangeReason = "复测长度修正"
	updated, err = fixture.Records.Update(ctx, record.ID, second)
	testsupport.RequireNoError(t, err)
	if updated.Version != 3 {
		t.Fatalf("二次修改后版本号应为 3，实际 %d", updated.Version)
	}

	revisions, err = fixture.Records.Revisions(ctx, record.ID)
	testsupport.RequireNoError(t, err)
	if len(revisions.Items) != 3 {
		t.Fatalf("期望 3 个版本，实际 %d", len(revisions.Items))
	}
	// 列表最新版本在前。
	if revisions.Items[0].Version != 3 || revisions.Items[2].Version != 1 {
		t.Fatalf("版本应按倒序返回，实际 %d,%d,%d",
			revisions.Items[0].Version, revisions.Items[1].Version, revisions.Items[2].Version)
	}

	latest := revisions.Items[0]
	v1 := revisions.Items[2]
	if latest.ChangedBy != "张强" || latest.ChangeReason != "复测长度修正" {
		t.Fatalf("最新版本留痕信息不正确: %+v", latest)
	}
	// 任意两个版本可以对比出字段差异，历史版本保存的是当次修改后的数值。
	if latest.Snapshot.SludgeVolumeM3 != 20 || latest.Snapshot.LengthM != 130 {
		t.Fatalf("v3 快照数值不正确: %+v", latest.Snapshot)
	}
	// v1 保留首次录入的原始数值（fixture 录入：LengthM=100, Sludge=10）。
	if v1.Snapshot.SludgeVolumeM3 != 10 || v1.Snapshot.LengthM != 100 {
		t.Fatalf("v1 应保留首次录入的旧数值: %+v", v1.Snapshot)
	}
	// v2 是第一次修改后的数值：清淤量 20，长度按 recordRequest 变为 80。
	if v2 := revisions.Items[1]; v2.Snapshot.SludgeVolumeM3 != 20 || v2.Snapshot.LengthM != 80 {
		t.Fatalf("v2 应保留第一次修改后的数值: %+v", v2.Snapshot)
	}
}

func TestUpdateRecordNoopIsRejected(t *testing.T) {
	ctx := context.Background()
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "无变化任务")
	record := fixture.CreateRecord(t, task.ID, 10)

	// 用与当前内容完全一致的请求提交（fixture 录入：LengthM=100, Sludge=10, Water=30, 人数=5）。
	req := cleaningrecord.SaveRequest{
		TaskID: task.ID, CleanedAt: date.Today().AddDays(-1),
		LengthM: 100, SludgeVolumeM3: 10, WaterVolumeM3: 30, PersonnelCount: 5,
		Method: cleaningtask.MethodHighPressure, Weather: cleaningrecord.WeatherSunny,
		RecorderName: "测试记录人",
	}
	_, err := fixture.Records.Update(ctx, record.ID, updateRequest(req))
	testsupport.RequireAppError(t, err, httpx.CodeValidation)

	revisions, err := fixture.Records.Revisions(ctx, record.ID)
	testsupport.RequireNoError(t, err)
	if len(revisions.Items) != 1 {
		t.Fatalf("无变化的提交不应产生新版本，实际版本数 %d", len(revisions.Items))
	}
}

func TestTotalsReflectLatestVersion(t *testing.T) {
	ctx := context.Background()
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "统计口径任务")
	record := fixture.CreateRecord(t, task.ID, 10)
	fixture.CreateRecord(t, task.ID, 15)

	// 修改第一条：10 -> 40，统计应立即按最新版本变为 55。
	req := recordRequest(task.ID)
	req.SludgeVolumeM3 = 40
	req.LengthM = 200
	_, err := fixture.Records.Update(ctx, record.ID, updateRequest(req))
	testsupport.RequireNoError(t, err)

	totals, err := fixture.Records.TotalsByTask(ctx, task.ID)
	testsupport.RequireNoError(t, err)
	if totals.RecordCount != 2 {
		t.Fatalf("记录条数仍应为 2，实际 %d", totals.RecordCount)
	}
	if totals.SludgeVolumeM3 != 55 {
		t.Fatalf("清淤量应按最新版本合计 55，实际 %v", totals.SludgeVolumeM3)
	}
	if totals.CleanedLengthM != 300 {
		t.Fatalf("清淤长度应按最新版本合计 300，实际 %v", totals.CleanedLengthM)
	}
}

func TestDeleteRecordBlockedWhenReferencedByAcceptance(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "被验收引用的任务")
	record := fixture.CreateRecord(t, task.ID, 15)
	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Complete(context.Background(), task.ID)))

	request := testsupport.ReworkRequest(task.ID)
	request.CleaningRecordID = &record.ID
	_, err := fixture.Acceptances.Create(context.Background(), request)
	testsupport.RequireNoError(t, err)

	// 验收需整改会把任务退回清淤中，此时任务环节仍开放，但记录已被验收引用不能删除 / 修改。
	err = fixture.Records.Delete(context.Background(), record.ID)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)

	updateReq := recordRequest(task.ID)
	updateReq.SludgeVolumeM3 = 99
	_, err = fixture.Records.Update(context.Background(), record.ID, updateRequest(updateReq))
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

func TestDeleteRecordAlsoRemovesRevisions(t *testing.T) {
	ctx := context.Background()
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "删除履历任务")
	record := fixture.CreateRecord(t, task.ID, 10)

	req := recordRequest(task.ID)
	req.SludgeVolumeM3 = 22
	_, err := fixture.Records.Update(ctx, record.ID, updateRequest(req))
	testsupport.RequireNoError(t, err)

	testsupport.RequireNoError(t, fixture.Records.Delete(ctx, record.ID))
	revisions, err := fixture.Records.Revisions(ctx, record.ID)
	testsupport.RequireAppError(t, err, httpx.CodeNotFound)
	_ = revisions
}

func TestDetailReportsEditableWindow(t *testing.T) {
	ctx := context.Background()
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "窗口状态任务")
	record := fixture.CreateRecord(t, task.ID, 10)

	detail, err := fixture.Records.Detail(ctx, record.ID)
	testsupport.RequireNoError(t, err)
	if !detail.Editable || detail.LockedReason != "" {
		t.Fatalf("清淤中任务应处于可修改窗口，实际 editable=%v reason=%q", detail.Editable, detail.LockedReason)
	}
	if detail.RevisionCount != 1 {
		t.Fatalf("版本数应为 1，实际 %d", detail.RevisionCount)
	}

	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Complete(ctx, task.ID)))
	detail, err = fixture.Records.Detail(ctx, record.ID)
	testsupport.RequireNoError(t, err)
	if detail.Editable || detail.LockedReason == "" {
		t.Fatalf("报验后窗口应关闭并给出原因，实际 editable=%v reason=%q", detail.Editable, detail.LockedReason)
	}
}

func TestRecordTotalsAggregateByTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "汇总任务")
	fixture.CreateRecord(t, task.ID, 10)
	fixture.CreateRecord(t, task.ID, 15)

	totals, err := fixture.Records.TotalsByTask(context.Background(), task.ID)
	testsupport.RequireNoError(t, err)
	if totals.RecordCount != 2 {
		t.Fatalf("期望记录条数 2，实际 %d", totals.RecordCount)
	}
	if totals.SludgeVolumeM3 != 25 {
		t.Fatalf("期望清淤量合计 25，实际 %v", totals.SludgeVolumeM3)
	}
	if totals.CleanedLengthM != 200 {
		t.Fatalf("期望清淤长度合计 200，实际 %v", totals.CleanedLengthM)
	}
}

func TestListRecordsFiltersByTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	first := fixture.CreateTask(t, fixture.Segment.ID, "任务一")
	second := fixture.CreateTask(t, fixture.Segment.ID, "任务二")
	fixture.CreateRecord(t, first.ID, 10)
	fixture.CreateRecord(t, second.ID, 20)

	items, total, err := fixture.Records.List(context.Background(), cleaningrecord.ListQuery{
		TaskID: first.ID,
		Page:   httpx.PageQuery{Page: 1, PageSize: 10},
	})
	testsupport.RequireNoError(t, err)
	if total != 1 || len(items) != 1 {
		t.Fatalf("期望筛出 1 条清淤记录，实际 total=%d len=%d", total, len(items))
	}
	if items[0].Task == nil || items[0].Task.ID != first.ID {
		t.Fatalf("列表项应带出所属任务信息，实际 %+v", items[0].Task)
	}
}

func ignoreTask(_ *cleaningtask.CleaningTask, err error) error {
	return err
}
