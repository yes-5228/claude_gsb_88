package pipesegment_test

import (
	"context"
	"testing"

	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/testsupport"
)

func segmentRequest(code, district string) pipesegment.SaveRequest {
	return pipesegment.SaveRequest{
		Code:         code,
		Name:         "管段 " + code,
		District:     district,
		RoadName:     "中山北路",
		PipeType:     pipesegment.TypeRainwater,
		Material:     "concrete",
		DiameterMm:   800,
		LengthM:      156.5,
		DepthM:       3.2,
		StartManhole: "Y1-08",
		EndManhole:   "Y1-12",
		BuildYear:    2012,
		OwnerUnit:    "市政排水管理处",
	}
}

func TestCreateSegmentRejectsDuplicatedCode(t *testing.T) {
	fixture := testsupport.NewFixture(t)

	_, err := fixture.Segments.Create(context.Background(), segmentRequest("PS-TEST-001", "城西片区"))
	appErr := testsupport.RequireAppError(t, err, httpx.CodeConflict)
	if appErr.Status != 409 {
		t.Fatalf("期望 HTTP 状态码 409，实际 %d", appErr.Status)
	}
}

func TestCreateSegmentRejectsUnknownPipeType(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	request := segmentRequest("PS-TEST-002", "城东片区")
	request.PipeType = "stormwater"

	_, err := fixture.Segments.Create(context.Background(), request)
	testsupport.RequireAppError(t, err, httpx.CodeValidation)
}

func TestCreateSegmentDefaultsStatusToNormal(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	segment := fixture.CreateSegment(t, "PS-TEST-003", "城东片区")

	if segment.Status != pipesegment.StatusNormal {
		t.Fatalf("期望默认状态为 %s，实际 %s", pipesegment.StatusNormal, segment.Status)
	}
}

func TestDeleteSegmentBlockedWhenReferencedByTask(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	fixture.CreateTask(t, fixture.Segment.ID, "引用管段的任务")

	err := fixture.Segments.Delete(context.Background(), fixture.Segment.ID)
	testsupport.RequireAppError(t, err, httpx.CodeConflict)

	if _, err := fixture.Segments.FindByID(context.Background(), fixture.Segment.ID); err != nil {
		t.Fatalf("管段不应被删除: %v", err)
	}
}

func TestDeleteSegmentSucceedsWhenNotReferenced(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	segment := fixture.CreateSegment(t, "PS-TEST-004", "城西片区")

	testsupport.RequireNoError(t, fixture.Segments.Delete(context.Background(), segment.ID))

	err := fixture.Segments.Delete(context.Background(), segment.ID)
	testsupport.RequireAppError(t, err, httpx.CodeNotFound)
}

func TestListSegmentsFiltersByDistrictAndKeyword(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	fixture.CreateSegment(t, "PS-TEST-005", "城西片区")
	fixture.CreateSegment(t, "PS-TEST-006", "城南片区")

	byDistrict, total, err := fixture.Segments.List(context.Background(), pipesegment.ListQuery{
		District: "城西片区",
		Page:     httpx.PageQuery{Page: 1, PageSize: 10},
	})
	testsupport.RequireNoError(t, err)
	if total != 1 || len(byDistrict) != 1 {
		t.Fatalf("期望按片区筛出 1 条管段，实际 total=%d len=%d", total, len(byDistrict))
	}

	byKeyword, _, err := fixture.Segments.List(context.Background(), pipesegment.ListQuery{
		Keyword: "ps-TEST-006",
		Page:    httpx.PageQuery{Page: 1, PageSize: 10},
	})
	testsupport.RequireNoError(t, err)
	if len(byKeyword) != 1 || byKeyword[0].Code != "PS-TEST-006" {
		t.Fatalf("期望按编号关键字忽略大小写命中 PS-TEST-006，实际 %+v", byKeyword)
	}
}

func TestDetailReturnsTaskStats(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	fixture.CreateTask(t, fixture.Segment.ID, "任务一")
	fixture.CreateTask(t, fixture.Segment.ID, "任务二")

	detail, err := fixture.Segments.Detail(context.Background(), fixture.Segment.ID)
	testsupport.RequireNoError(t, err)

	if detail.TaskStats.Total != 2 || detail.TaskStats.Pending != 2 {
		t.Fatalf("期望 2 条待开工任务，实际 %+v", detail.TaskStats)
	}
	if len(detail.RecentTasks) != 2 {
		t.Fatalf("期望最近任务列表包含 2 条记录，实际 %d", len(detail.RecentTasks))
	}
}

func TestHistoryIncludesAcceptanceResult(t *testing.T) {
	fixture := testsupport.NewFixture(t)
	task := fixture.TaskReadyForAcceptance(t, fixture.Segment.ID, "清淤任务")
	_, err := fixture.Acceptances.Create(context.Background(), testsupport.PassRequest(task.ID, 90))
	testsupport.RequireNoError(t, err)

	items, err := fixture.Segments.History(context.Background(), fixture.Segment.ID)
	testsupport.RequireNoError(t, err)
	if len(items) != 1 {
		t.Fatalf("期望 1 条清淤履历，实际 %d", len(items))
	}
	if items[0].AcceptanceResult != "pass" {
		t.Fatalf("期望履历中带出验收结论 pass，实际 %q", items[0].AcceptanceResult)
	}
	if items[0].RecordCount != 1 {
		t.Fatalf("期望履历中记录条数为 1，实际 %d", items[0].RecordCount)
	}
}
