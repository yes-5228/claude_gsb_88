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

// updateRequest 在录入请求的基础上补齐修改人与修改原因（修改留痕必填）。
func updateRequest(taskID uint) cleaningrecord.UpdateRequest {
	return cleaningrecord.UpdateRequest{
		SaveRequest:  recordRequest(taskID),
		EditorName:   "李伟",
		ChangeReason: "现场复核后修正数据",
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

	request := updateRequest(task.ID)
	request.SludgeVolumeM3 = 20
	_, err := fixture.Records.Update(context.Background(), record.ID, request)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)

	// 窗口关闭后绕过页面提交也会被拦下，且不会留下任何留痕。
	versions, verr := fixture.Records.Versions(context.Background(), record.ID)
	testsupport.RequireNoError(t, verr)
	if versions.CurrentVersion != 1 {
		t.Fatalf("窗口关闭后的修改不应产生留痕，实际版本数 %d", versions.CurrentVersion)
	}
}

func TestUpdateRecordRejectsChangingTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	first := fixture.CreateTask(t, fixture.Segment.ID, "任务一")
	second := fixture.CreateTask(t, fixture.Segment.ID, "任务二")
	record := fixture.CreateRecord(t, first.ID, 9)

	request := updateRequest(second.ID)
	_, err := fixture.Records.Update(context.Background(), record.ID, request)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
}

// 每次修改都要留痕：修改前的数值、修改人、时间与原因完整保存，当前版本号随之递增。
func TestUpdateRecordKeepsRevisionTrace(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "留痕校验任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	request := updateRequest(task.ID)
	request.SludgeVolumeM3 = 21.5
	request.EditorName = "王芳"
	request.ChangeReason = "按吸污车称重小票复核后修正清淤量"
	updated, err := fixture.Records.Update(context.Background(), record.ID, request)
	testsupport.RequireNoError(t, err)
	if updated.SludgeVolumeM3 != 21.5 {
		t.Fatalf("期望清淤量更新为 21.5，实际 %v", updated.SludgeVolumeM3)
	}

	versions, err := fixture.Records.Versions(context.Background(), record.ID)
	testsupport.RequireNoError(t, err)
	if versions.CurrentVersion != 2 || len(versions.Versions) != 2 {
		t.Fatalf("期望当前版本为 v2，实际 current=%d len=%d", versions.CurrentVersion, len(versions.Versions))
	}
	first := versions.Versions[0]
	if first.Current || first.SludgeVolumeM3 != 9 {
		t.Fatalf("v1 应保存修改前的数值 9，实际 %+v", first)
	}
	if first.EditorName != "王芳" || first.ChangeReason != "按吸污车称重小票复核后修正清淤量" || first.ChangedAt == nil {
		t.Fatalf("v1 留痕应包含修改人、修改原因与修改时间，实际 %+v", first)
	}
	latest := versions.Versions[1]
	if !latest.Current || latest.SludgeVolumeM3 != 21.5 {
		t.Fatalf("v2 应为当前版本且数值为 21.5，实际 %+v", latest)
	}
}

// 连续修改会形成完整版本链，任意两个版本都能取到当时的数值。
func TestUpdateRecordTracesEveryChange(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "多版本校验任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	first := updateRequest(task.ID)
	first.SludgeVolumeM3 = 11
	if _, err := fixture.Records.Update(context.Background(), record.ID, first); err != nil {
		testsupport.RequireNoError(t, err)
	}
	second := updateRequest(task.ID)
	second.SludgeVolumeM3 = 14
	if _, err := fixture.Records.Update(context.Background(), record.ID, second); err != nil {
		testsupport.RequireNoError(t, err)
	}

	versions, err := fixture.Records.Versions(context.Background(), record.ID)
	testsupport.RequireNoError(t, err)
	if versions.CurrentVersion != 3 || len(versions.Versions) != 3 {
		t.Fatalf("期望 3 个版本，实际 current=%d len=%d", versions.CurrentVersion, len(versions.Versions))
	}
	want := []float64{9, 11, 14}
	for i, version := range versions.Versions {
		if version.Version != i+1 || version.SludgeVolumeM3 != want[i] {
			t.Fatalf("v%d 数值应为 %v，实际 %+v", i+1, want[i], version)
		}
	}
	if !versions.Versions[2].Current {
		t.Fatal("最后一个版本应标记为当前版本")
	}
}

// 修改人与修改原因是留痕的必填项，缺失时拒绝修改且不留任何痕迹。
func TestUpdateRecordRequiresEditorAndReason(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "留痕必填校验任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	missingEditor := updateRequest(task.ID)
	missingEditor.EditorName = " "
	if _, err := fixture.Records.Update(context.Background(), record.ID, missingEditor); err != nil {
		testsupport.RequireAppError(t, err, httpx.CodeValidation)
	}

	missingReason := updateRequest(task.ID)
	missingReason.ChangeReason = ""
	if _, err := fixture.Records.Update(context.Background(), record.ID, missingReason); err != nil {
		testsupport.RequireAppError(t, err, httpx.CodeValidation)
	}

	versions, err := fixture.Records.Versions(context.Background(), record.ID)
	testsupport.RequireNoError(t, err)
	if versions.CurrentVersion != 1 {
		t.Fatalf("校验失败不应产生留痕，实际版本数 %d", versions.CurrentVersion)
	}
	reloaded, err := fixture.Records.FindByID(context.Background(), record.ID)
	testsupport.RequireNoError(t, err)
	if reloaded.SludgeVolumeM3 != 9 {
		t.Fatalf("校验失败时数值不应变化，期望 9，实际 %v", reloaded.SludgeVolumeM3)
	}
}

// 修改与留痕必须同时成立：留痕写不进去时，数值也不能变。
func TestUpdateRecordRollsBackWhenRevisionFails(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "原子性校验任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	// 人为制造留痕写入失败：删掉留痕表后，任何修改都必须整体失败并回滚。
	testsupport.RequireNoError(t, fixture.DB.Migrator().DropTable("cleaning_record_revisions"))

	request := updateRequest(task.ID)
	request.SludgeVolumeM3 = 30
	_, err := fixture.Records.Update(context.Background(), record.ID, request)
	testsupport.RequireAppError(t, err, httpx.CodeInternal)

	reloaded, loadErr := fixture.Records.FindByID(context.Background(), record.ID)
	testsupport.RequireNoError(t, loadErr)
	if reloaded.SludgeVolumeM3 != 9 {
		t.Fatalf("留痕失败时数值不应变化，期望 9，实际 %v", reloaded.SludgeVolumeM3)
	}
}

// 被验收记录引用的清淤记录只能查看历史，即使任务退回清淤中也不能再修改。
func TestUpdateRecordBlockedWhenReferencedByAcceptance(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "验收引用校验任务")
	record := fixture.CreateRecord(t, task.ID, 15)
	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Complete(context.Background(), task.ID)))

	request := testsupport.ReworkRequest(task.ID)
	request.CleaningRecordID = &record.ID
	_, err := fixture.Acceptances.Create(context.Background(), request)
	testsupport.RequireNoError(t, err)

	// 验收需整改会把任务退回清淤中，窗口看似重新开放，
	// 但记录已被验收引用，仍然只能查看历史版本。
	_, err = fixture.Records.Update(context.Background(), record.ID, updateRequest(task.ID))
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)

	detail, derr := fixture.Records.Detail(context.Background(), record.ID)
	testsupport.RequireNoError(t, derr)
	if detail.EditWindow.Open || detail.EditWindow.Reason == "" {
		t.Fatalf("被验收引用的记录窗口应关闭并给出原因，实际 %+v", detail.EditWindow)
	}
}

// 所有统计按最新版本计算：修改提交后，任务汇总立即反映新数值。
func TestTotalsReflectLatestVersionAfterUpdate(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "汇总联动任务")
	record := fixture.CreateRecord(t, task.ID, 10)

	request := updateRequest(task.ID)
	request.SludgeVolumeM3 = 26
	request.LengthM = 150
	_, err := fixture.Records.Update(context.Background(), record.ID, request)
	testsupport.RequireNoError(t, err)

	totals, err := fixture.Records.TotalsByTask(context.Background(), task.ID)
	testsupport.RequireNoError(t, err)
	if totals.SludgeVolumeM3 != 26 || totals.CleanedLengthM != 150 {
		t.Fatalf("汇总应按最新版本计算，实际 %+v", totals)
	}
}

// 详情返回修改窗口状态：窗口随任务环节推进而关闭。
func TestDetailReflectsEditWindow(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "窗口状态任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	detail, err := fixture.Records.Detail(context.Background(), record.ID)
	testsupport.RequireNoError(t, err)
	if !detail.EditWindow.Open || detail.CurrentVersion != 1 {
		t.Fatalf("清淤中的任务窗口应开放且为 v1，实际 %+v / v%d", detail.EditWindow, detail.CurrentVersion)
	}

	testsupport.RequireNoError(t, ignoreTask(fixture.Tasks.Complete(context.Background(), task.ID)))
	detail, err = fixture.Records.Detail(context.Background(), record.ID)
	testsupport.RequireNoError(t, err)
	if detail.EditWindow.Open || detail.EditWindow.Reason == "" {
		t.Fatalf("完工报验后窗口应关闭并给出原因，实际 %+v", detail.EditWindow)
	}
}

// 删除记录时留痕在同一事务内一并清除。
func TestDeleteRecordRemovesRevisions(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.CreateTask(t, fixture.Segment.ID, "删除清理任务")
	record := fixture.CreateRecord(t, task.ID, 9)

	if _, err := fixture.Records.Update(context.Background(), record.ID, updateRequest(task.ID)); err != nil {
		testsupport.RequireNoError(t, err)
	}
	testsupport.RequireNoError(t, fixture.Records.Delete(context.Background(), record.ID))

	var count int64
	testsupport.RequireNoError(t, fixture.DB.Table("cleaning_record_revisions").
		Where("record_id = ?", record.ID).
		Count(&count).Error)
	if count != 0 {
		t.Fatalf("删除记录后留痕应一并清除，实际剩余 %d 条", count)
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

	// 验收需整改会把任务退回清淤中，此时记录本身可编辑，但已被验收引用不能删除
	err = fixture.Records.Delete(context.Background(), record.ID)
	testsupport.RequireAppError(t, err, httpx.CodeInvalidState)
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
