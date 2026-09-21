// Package testsupport 为单元测试提供内存数据库与业务服务装配。
//
// 测试使用内存 SQLite，好处是不依赖 PostgreSQL 即可覆盖全部业务规则，
// 同时与生产环境共用同一套 model、repository 与 service 代码。
package testsupport

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/drainage/desilting/internal/database"
	"github.com/drainage/desilting/internal/modules/acceptance"
	"github.com/drainage/desilting/internal/modules/cleaningrecord"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/shared/date"
)

// NewDB 创建仅供本次测试使用的内存 SQLite 数据库，并完成表结构迁移。
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	// 测试名可能包含斜杠，先做一次安全替换，避免被当成文件路径片段。
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", safeName)), &gorm.Config{
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取数据库连接失败: %v", err)
	}
	// 内存库必须限制为单个连接，否则不同连接会看到不同的数据库。
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := database.Migrate(db); err != nil {
		t.Fatalf("迁移表结构失败: %v", err)
	}
	return db
}

// Services 一套装配完成的业务服务，等价于 router 中的生产装配方式。
type Services struct {
	Segments    *pipesegment.Service
	Tasks       *cleaningtask.Service
	Records     *cleaningrecord.Service
	Acceptances *acceptance.Service
}

// NewServices 按生产环境的依赖顺序装配服务。
func NewServices(db *gorm.DB) *Services {
	segments := pipesegment.NewService(pipesegment.NewRepository(db))
	tasks := cleaningtask.NewService(cleaningtask.NewRepository(db), segments)
	records := cleaningrecord.NewService(cleaningrecord.NewRepository(db), tasks)
	acceptances := acceptance.NewService(acceptance.NewRepository(db), tasks, segments, records)
	return &Services{Segments: segments, Tasks: tasks, Records: records, Acceptances: acceptances}
}

// Fixture 内存库 + 服务 + 默认管段的组合，方便测试用例直接使用。
type Fixture struct {
	*Services
	DB      *gorm.DB
	Segment *pipesegment.PipeSegment
}

// NewFixture 建库、装配服务并创建一条默认管段。
func NewFixture(t *testing.T) *Fixture {
	t.Helper()
	db := NewDB(t)
	services := NewServices(db)
	segment := services.CreateSegment(t, "PS-TEST-001", "城东片区")
	return &Fixture{Services: services, DB: db, Segment: segment}
}

// CreateSegment 创建一条管段。
func (s *Services) CreateSegment(t *testing.T, code, district string) *pipesegment.PipeSegment {
	t.Helper()
	segment, err := s.Segments.Create(context.Background(), pipesegment.SaveRequest{
		Code:         code,
		Name:         "测试管段 " + code,
		District:     district,
		RoadName:     "测试道路",
		PipeType:     pipesegment.TypeRainwater,
		Material:     "concrete",
		DiameterMm:   600,
		LengthM:      120,
		DepthM:       2.5,
		StartManhole: "Y1-01",
		EndManhole:   "Y1-05",
		BuildYear:    2015,
	})
	if err != nil {
		t.Fatalf("创建测试管段失败: %v", err)
	}
	return segment
}

// CreateTask 创建一条待开工的清淤任务（计划开始日期为 3 天前）。
func (s *Services) CreateTask(t *testing.T, segmentID uint, title string) *cleaningtask.CleaningTask {
	t.Helper()
	task, err := s.Tasks.Create(context.Background(), cleaningtask.SaveRequest{
		Title:         title,
		PipeSegmentID: segmentID,
		Priority:      cleaningtask.PriorityNormal,
		Source:        cleaningtask.SourcePlan,
		Method:        cleaningtask.MethodHighPressure,
		PlanStartDate: date.Today().AddDays(-3),
		PlanEndDate:   date.Today().AddDays(1),
		TeamName:      "测试班组",
		LeaderName:    "测试负责人",
		LeaderPhone:   "0571-88888888",
	})
	if err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}
	return task
}

// CreateRecord 为任务录入一条清淤记录（清淤日期为 1 天前）。
func (s *Services) CreateRecord(t *testing.T, taskID uint, sludge float64) *cleaningrecord.CleaningRecord {
	t.Helper()
	record, err := s.Records.Create(context.Background(), cleaningrecord.SaveRequest{
		TaskID:         taskID,
		CleanedAt:      date.Today().AddDays(-1),
		LengthM:        100,
		SludgeVolumeM3: sludge,
		WaterVolumeM3:  30,
		PersonnelCount: 5,
		Method:         cleaningtask.MethodHighPressure,
		Weather:        cleaningrecord.WeatherSunny,
		RecorderName:   "测试记录人",
	})
	if err != nil {
		t.Fatalf("录入测试清淤记录失败: %v", err)
	}
	return record
}

// TaskReadyForAcceptance 创建一条已完工待验收的任务（含一条清淤记录）。
func (s *Services) TaskReadyForAcceptance(t *testing.T, segmentID uint, title string) *cleaningtask.CleaningTask {
	t.Helper()
	task := s.CreateTask(t, segmentID, title)
	s.CreateRecord(t, task.ID, 12.5)
	if _, err := s.Tasks.Complete(context.Background(), task.ID); err != nil {
		t.Fatalf("提交完工报验失败: %v", err)
	}
	return s.Reload(t, task.ID)
}

// Reload 重新读取任务的最新状态。
func (s *Services) Reload(t *testing.T, taskID uint) *cleaningtask.CleaningTask {
	t.Helper()
	task, err := s.Tasks.FindByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	return task
}

// PassRequest 构造一份验收合格的请求。
func PassRequest(taskID uint, score int) acceptance.SaveRequest {
	return acceptance.SaveRequest{
		TaskID:           taskID,
		AcceptedAt:       date.Today(),
		InspectorName:    "测试验收人",
		InspectorOrg:     "测试验收单位",
		Result:           acceptance.ResultPass,
		Score:            score,
		ResidualSludgeMm: 10,
	}
}

// ReworkRequest 构造一份验收需整改的请求。
func ReworkRequest(taskID uint) acceptance.SaveRequest {
	deadline := date.Today().AddDays(3)
	return acceptance.SaveRequest{
		TaskID:           taskID,
		AcceptedAt:       date.Today(),
		InspectorName:    "测试验收人",
		InspectorOrg:     "测试验收单位",
		Result:           acceptance.ResultRework,
		Score:            50,
		ResidualSludgeMm: 40,
		Issues:           "残留淤积厚度超出验收标准",
		RectifyDeadline:  &deadline,
	}
}
