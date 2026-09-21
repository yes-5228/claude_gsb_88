package database

import (
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/modules/acceptance"
	"github.com/drainage/desilting/internal/modules/cleaningrecord"
	"github.com/drainage/desilting/internal/modules/cleaningtask"
	"github.com/drainage/desilting/internal/modules/pipesegment"
	"github.com/drainage/desilting/internal/shared/date"
)

// Seed 写入演示数据，便于首次启动后直接体验完整业务链路。
//
// 只在管段台账为空时执行，因此重复启动不会产生重复数据。
// 演示数据覆盖了全部任务状态：待开工、清淤中、待验收、已验收、已取消，
// 以及"验收需整改 -> 整改完成待复验"的中间态。
func Seed(db *gorm.DB, logger *slog.Logger) error {
	var segmentCount int64
	if err := db.Model(&pipesegment.PipeSegment{}).Count(&segmentCount).Error; err != nil {
		return err
	}
	if segmentCount > 0 {
		logger.Info("业务数据已存在，跳过演示数据初始化", "segments", segmentCount)
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		today := date.Today()

		segments := []pipesegment.PipeSegment{
			{
				Code: "PS-Y-2021-001", Name: "中山北路雨水主干管", District: "城东片区", RoadName: "中山北路",
				PipeType: pipesegment.TypeRainwater, Material: "concrete", DiameterMm: 800, LengthM: 156.5, DepthM: 3.2,
				StartManhole: "Y1-08", EndManhole: "Y1-12", BuildYear: 2012, OwnerUnit: "市政排水管理处",
				Status: pipesegment.StatusNormal, CleanedTimes: 1, LastCleanedAt: ptrDate(today.AddDays(-28)),
				Remark: "承担中山北路北段雨水排放，汛期前需完成清淤",
			},
			{
				Code: "PS-Y-2021-002", Name: "中山北路雨水支管", District: "城东片区", RoadName: "中山北路",
				PipeType: pipesegment.TypeRainwater, Material: "hdpe", DiameterMm: 400, LengthM: 88, DepthM: 2.1,
				StartManhole: "Y1-12", EndManhole: "Y1-16", BuildYear: 2015, OwnerUnit: "市政排水管理处",
				Status: pipesegment.StatusAttention, CleanedTimes: 1, LastCleanedAt: ptrDate(today.AddDays(-17)),
				Remark: "管段存在错口，清淤后仍有少量积水",
			},
			{
				Code: "PS-W-2018-014", Name: "解放东路污水干管", District: "城西片区", RoadName: "解放东路",
				PipeType: pipesegment.TypeSewage, Material: "concrete", DiameterMm: 1000, LengthM: 320, DepthM: 4.5,
				StartManhole: "W2-03", EndManhole: "W2-11", BuildYear: 2009, OwnerUnit: "市政排水管理处",
				Status: pipesegment.StatusNormal, CleanedTimes: 1, LastCleanedAt: ptrDate(today.AddDays(-21)),
			},
			{
				Code: "PS-W-2018-015", Name: "解放东路污水支管", District: "城西片区", RoadName: "解放东路",
				PipeType: pipesegment.TypeSewage, Material: "ductile_iron", DiameterMm: 500, LengthM: 120, DepthM: 2.8,
				StartManhole: "W2-11", EndManhole: "W2-15", BuildYear: 2014, OwnerUnit: "市政排水管理处",
				Status: pipesegment.StatusBlocked, Remark: "W2-12 井段曾发现建筑垃圾，需重点复查",
			},
			{
				Code: "PS-H-2020-007", Name: "人民广场合流管", District: "城南片区", RoadName: "人民广场环路",
				PipeType: pipesegment.TypeCombined, Material: "concrete", DiameterMm: 1200, LengthM: 210, DepthM: 5,
				StartManhole: "H3-01", EndManhole: "H3-06", BuildYear: 2011, OwnerUnit: "人民广场管理办公室",
				Status: pipesegment.StatusAttention, Remark: "H3-04 井段存在树根侵入",
			},
			{
				Code: "PS-Y-2022-033", Name: "长江南路雨水管", District: "城南片区", RoadName: "长江南路",
				PipeType: pipesegment.TypeRainwater, Material: "hdpe", DiameterMm: 600, LengthM: 175, DepthM: 2.6,
				StartManhole: "Y4-05", EndManhole: "Y4-11", BuildYear: 2018, OwnerUnit: "城南片区养护站",
				Status: pipesegment.StatusNormal,
			},
			{
				Code: "PS-W-2019-021", Name: "长江南路污水管", District: "城南片区", RoadName: "长江南路",
				PipeType: pipesegment.TypeSewage, Material: "concrete", DiameterMm: 600, LengthM: 190, DepthM: 3.4,
				StartManhole: "W4-02", EndManhole: "W4-08", BuildYear: 2010, OwnerUnit: "城南片区养护站",
				Status: pipesegment.StatusNormal,
			},
			{
				Code: "PS-Y-2023-046", Name: "滨江大道雨水管", District: "城东片区", RoadName: "滨江大道",
				PipeType: pipesegment.TypeRainwater, Material: "grp", DiameterMm: 1000, LengthM: 265, DepthM: 3.8,
				StartManhole: "Y6-01", EndManhole: "Y6-09", BuildYear: 2020, OwnerUnit: "滨江新区建设指挥部",
				Status: pipesegment.StatusNormal,
			},
			{
				Code: "PS-H-2020-009", Name: "人民广场合流支管", District: "城南片区", RoadName: "人民广场西路",
				PipeType: pipesegment.TypeCombined, Material: "concrete", DiameterMm: 600, LengthM: 96, DepthM: 3.1,
				StartManhole: "H3-06", EndManhole: "H3-09", BuildYear: 2011, OwnerUnit: "人民广场管理办公室",
				Status: pipesegment.StatusNormal, Remark: "尚未安排过清淤",
			},
			{
				Code: "PS-Y-2022-035", Name: "长江南路雨水支管", District: "城南片区", RoadName: "长江支路",
				PipeType: pipesegment.TypeRainwater, Material: "pvc", DiameterMm: 300, LengthM: 64, DepthM: 1.8,
				StartManhole: "Y4-11", EndManhole: "Y4-13", BuildYear: 2018, OwnerUnit: "城南片区养护站",
				Status: pipesegment.StatusNormal, Remark: "尚未安排过清淤",
			},
		}
		if err := tx.Create(&segments).Error; err != nil {
			return err
		}

		segmentID := make(map[string]uint, len(segments))
		for i := range segments {
			segmentID[segments[i].Code] = segments[i].ID
		}

		tasks := []cleaningtask.CleaningTask{
			{
				Code:  "QX" + today.AddDays(-32).Format("20060102") + "-0001",
				Title: "中山北路雨水主干管汛前清淤", PipeSegmentID: segmentID["PS-Y-2021-001"],
				Priority: cleaningtask.PriorityHigh, Source: cleaningtask.SourcePlan, Method: cleaningtask.MethodHighPressure,
				PlanStartDate: today.AddDays(-32), PlanEndDate: today.AddDays(-28),
				TeamName: "城东养护一班", LeaderName: "李伟", LeaderPhone: "0571-88123456",
				Status: cleaningtask.StatusAccepted, Description: "汛期前完成主干管清淤，重点清理 Y1-10 井段淤积",
				StartedAt: stamp(today.AddDays(-32), 8), FinishedAt: stamp(today.AddDays(-27), 17),
				AcceptedAt: stamp(today.AddDays(-25), 10),
			},
			{
				Code:  "QX" + today.AddDays(-24).Format("20060102") + "-0001",
				Title: "解放东路污水干管年度清淤", PipeSegmentID: segmentID["PS-W-2018-014"],
				Priority: cleaningtask.PriorityNormal, Source: cleaningtask.SourcePlan, Method: cleaningtask.MethodWinch,
				PlanStartDate: today.AddDays(-24), PlanEndDate: today.AddDays(-20),
				TeamName: "城西养护二班", LeaderName: "张强", LeaderPhone: "0571-88234567",
				Status: cleaningtask.StatusAccepted, Description: "长距离污水干管，采用绞车牵引配合吸污车作业",
				StartedAt: stamp(today.AddDays(-21), 7), FinishedAt: stamp(today.AddDays(-19), 16),
				AcceptedAt: stamp(today.AddDays(-18), 9),
			},
			{
				Code:  "QX" + today.AddDays(-15).Format("20060102") + "-0001",
				Title: "解放东路污水支管淤堵清理", PipeSegmentID: segmentID["PS-W-2018-015"],
				Priority: cleaningtask.PriorityUrgent, Source: cleaningtask.SourceComplaint, Method: cleaningtask.MethodHighPressure,
				PlanStartDate: today.AddDays(-15), PlanEndDate: today.AddDays(-12),
				TeamName: "城西养护二班", LeaderName: "张强", LeaderPhone: "0571-88234567",
				Status: cleaningtask.StatusCompleted, Description: "市民反映井盖冒溢，排查后发现支管淤堵",
				StartedAt: stamp(today.AddDays(-14), 8), FinishedAt: stamp(today.AddDays(-11), 15),
			},
			{
				Code:  "QX" + today.AddDays(-6).Format("20060102") + "-0001",
				Title: "人民广场合流管树根侵入段清淤", PipeSegmentID: segmentID["PS-H-2020-007"],
				Priority: cleaningtask.PriorityHigh, Source: cleaningtask.SourceInspection, Method: cleaningtask.MethodGrab,
				PlanStartDate: today.AddDays(-6), PlanEndDate: today.AddDays(2),
				TeamName: "城南养护三班", LeaderName: "陈刚", LeaderPhone: "0571-88345678",
				Status: cleaningtask.StatusInProgress, Description: "结合管道检测结果，清理 H3-04 井段树根与沉积物",
				StartedAt: stamp(today.AddDays(-3), 8),
			},
			{
				Code:  "QX" + today.AddDays(-20).Format("20060102") + "-0001",
				Title: "中山北路雨水支管错口段清淤", PipeSegmentID: segmentID["PS-Y-2021-002"],
				Priority: cleaningtask.PriorityNormal, Source: cleaningtask.SourceInspection, Method: cleaningtask.MethodManual,
				PlanStartDate: today.AddDays(-20), PlanEndDate: today.AddDays(-16),
				TeamName: "城东养护一班", LeaderName: "李伟", LeaderPhone: "0571-88123456",
				Status: cleaningtask.StatusInProgress, Description: "验收发现管段错口未处理，退回整改后重新清淤",
				StartedAt: stamp(today.AddDays(-18), 8),
			},
			{
				Code:  "QX" + today.AddDays(3).Format("20060102") + "-0001",
				Title: "长江南路雨水管汛前清淤", PipeSegmentID: segmentID["PS-Y-2022-033"],
				Priority: cleaningtask.PriorityNormal, Source: cleaningtask.SourcePlan, Method: cleaningtask.MethodHighPressure,
				PlanStartDate: today.AddDays(3), PlanEndDate: today.AddDays(8),
				TeamName: "城南养护三班", LeaderName: "陈刚", LeaderPhone: "0571-88345678",
				Status: cleaningtask.StatusPending, Description: "按年度计划安排，待城东片区任务完成后进场",
			},
			{
				Code:  "QX" + today.AddDays(-10).Format("20060102") + "-0001",
				Title: "长江南路污水管清淤", PipeSegmentID: segmentID["PS-W-2019-021"],
				Priority: cleaningtask.PriorityLow, Source: cleaningtask.SourcePlan, Method: cleaningtask.MethodHighPressure,
				PlanStartDate: today.AddDays(-10), PlanEndDate: today.AddDays(-5),
				TeamName: "城南养护三班", LeaderName: "陈刚", LeaderPhone: "0571-88345678",
				Status: cleaningtask.StatusCancelled, CancelReason: "汛期调度调整，暂缓实施",
				Description: "与长江南路雨水管清淤同步实施",
			},
			{
				Code:  "QX" + today.AddDays(10).Format("20060102") + "-0001",
				Title: "滨江大道雨水管汛期专项清淤", PipeSegmentID: segmentID["PS-Y-2023-046"],
				Priority: cleaningtask.PriorityUrgent, Source: cleaningtask.SourceFlood, Method: cleaningtask.MethodRobot,
				PlanStartDate: today.AddDays(10), PlanEndDate: today.AddDays(16),
				TeamName: "滨江专项作业队", LeaderName: "周敏", LeaderPhone: "0571-88456789",
				Status: cleaningtask.StatusPending, Description: "汛期专项，采用管道机器人配合高压清洗",
			},
		}
		if err := tx.Create(&tasks).Error; err != nil {
			return err
		}

		taskID := make(map[string]uint, len(tasks))
		for i := range tasks {
			taskID[tasks[i].Code] = tasks[i].ID
		}

		records := []cleaningrecord.CleaningRecord{
			{
				Code:   "QJ" + today.AddDays(-30).Format("20060102") + "-0001",
				TaskID: taskID[tasks[0].Code], CleanedAt: today.AddDays(-30),
				LengthM: 80, SludgeVolumeM3: 12.5, WaterVolumeM3: 45, PersonnelCount: 6,
				Method: cleaningtask.MethodHighPressure, Equipment: "高压清洗车 2 台、吸污车 1 台", Weather: cleaningrecord.WeatherCloudy,
				SludgeDisposalSite: "城东污泥消纳中心", SafetyMeasures: "设置围挡与警示标志，下井前气体检测并持续通风",
				ProblemFound: "Y1-10 井段断面淤积约 30%", RecorderName: "李伟",
			},
			{
				Code:   "QJ" + today.AddDays(-28).Format("20060102") + "-0001",
				TaskID: taskID[tasks[0].Code], CleanedAt: today.AddDays(-28),
				LengthM: 76.5, SludgeVolumeM3: 9.8, WaterVolumeM3: 38, PersonnelCount: 5,
				Method: cleaningtask.MethodHighPressure, Equipment: "高压清洗车 2 台、吸污车 1 台", Weather: cleaningrecord.WeatherSunny,
				SludgeDisposalSite: "城东污泥消纳中心", SafetyMeasures: "设置围挡与警示标志，全程气体检测",
				RecorderName: "李伟", Remark: "复检后管内淤积厚度满足要求",
			},
			{
				Code:   "QJ" + today.AddDays(-21).Format("20060102") + "-0001",
				TaskID: taskID[tasks[1].Code], CleanedAt: today.AddDays(-21),
				LengthM: 320, SludgeVolumeM3: 42.6, WaterVolumeM3: 120, PersonnelCount: 8,
				Method: cleaningtask.MethodWinch, Equipment: "绞车 2 台、吸污车 2 台", Weather: cleaningrecord.WeatherSunny,
				SludgeDisposalSite: "城西污泥消纳中心", SafetyMeasures: "分段封堵导流，作业面设置安全通道",
				ProblemFound: "W2-07 井段沉积砂石较多，清淤两遍后达到要求", RecorderName: "张强",
			},
			{
				Code:   "QJ" + today.AddDays(-14).Format("20060102") + "-0001",
				TaskID: taskID[tasks[2].Code], CleanedAt: today.AddDays(-14),
				LengthM: 60, SludgeVolumeM3: 18.4, WaterVolumeM3: 36, PersonnelCount: 5,
				Method: cleaningtask.MethodHighPressure, Equipment: "高压清洗车 1 台、吸污车 1 台", Weather: cleaningrecord.WeatherLightRain,
				SludgeDisposalSite: "城西污泥消纳中心", SafetyMeasures: "雨天作业增设防滑措施，专人监护井口",
				ProblemFound: "W2-12 检查井内清出建筑垃圾约 0.6 m³", RecorderName: "王芳",
			},
			{
				Code:   "QJ" + today.AddDays(-12).Format("20060102") + "-0001",
				TaskID: taskID[tasks[2].Code], CleanedAt: today.AddDays(-12),
				LengthM: 60, SludgeVolumeM3: 15.2, WaterVolumeM3: 30, PersonnelCount: 5,
				Method: cleaningtask.MethodHighPressure, Equipment: "高压清洗车 1 台、吸污车 1 台", Weather: cleaningrecord.WeatherOvercast,
				SludgeDisposalSite: "城西污泥消纳中心", SafetyMeasures: "设置围挡与警示标志，全程气体检测",
				RecorderName: "王芳", Remark: "支管过流能力恢复正常，等待验收",
			},
			{
				Code:   "QJ" + today.AddDays(-3).Format("20060102") + "-0001",
				TaskID: taskID[tasks[3].Code], CleanedAt: today.AddDays(-3),
				LengthM: 110, SludgeVolumeM3: 26.3, WaterVolumeM3: 70, PersonnelCount: 7,
				Method: cleaningtask.MethodGrab, Equipment: "抓斗车 1 台、吸污车 1 台、管道检测机器人 1 台", Weather: cleaningrecord.WeatherOvercast,
				SludgeDisposalSite: "城南污泥消纳中心", SafetyMeasures: "井口设置三脚架与防坠装置，作业人员佩戴安全带",
				ProblemFound: "H3-04 井段存在树根侵入，已切除并记录待复检", RecorderName: "陈刚",
			},
			{
				Code:   "QJ" + today.AddDays(-17).Format("20060102") + "-0001",
				TaskID: taskID[tasks[4].Code], CleanedAt: today.AddDays(-17),
				LengthM: 88, SludgeVolumeM3: 7.6, WaterVolumeM3: 22, PersonnelCount: 4,
				Method: cleaningtask.MethodManual, Equipment: "人工清掏工具、吸污车 1 台", Weather: cleaningrecord.WeatherCloudy,
				SludgeDisposalSite: "城东污泥消纳中心", SafetyMeasures: "有限空间作业审批后实施，全程通风检测",
				ProblemFound: "管段错口约 5 cm，清淤后仍有少量积水", RecorderName: "刘洋",
			},
		}
		if err := tx.Create(&records).Error; err != nil {
			return err
		}

		acceptances := []acceptance.AcceptanceRecord{
			{
				Code:   "YS" + today.AddDays(-25).Format("20060102") + "-0001",
				TaskID: taskID[tasks[0].Code], CleaningRecordID: &records[1].ID,
				AcceptedAt: today.AddDays(-25), InspectorName: "赵敏", InspectorOrg: "市政排水管理处养护科",
				Result: acceptance.ResultPass, Score: 92, ResidualSludgeMm: 8,
				Remark: "管内淤积已清除，过水断面满足设计要求",
			},
			{
				Code:   "YS" + today.AddDays(-18).Format("20060102") + "-0001",
				TaskID: taskID[tasks[1].Code], CleaningRecordID: &records[2].ID,
				AcceptedAt: today.AddDays(-18), InspectorName: "赵敏", InspectorOrg: "市政排水管理处养护科",
				Result: acceptance.ResultPass, Score: 88, ResidualSludgeMm: 12,
				Remark: "长距离干管清淤质量良好，抽检三个井段均合格",
			},
			{
				Code:   "YS" + today.AddDays(-15).Format("20060102") + "-0001",
				TaskID: taskID[tasks[4].Code], CleaningRecordID: &records[6].ID,
				AcceptedAt: today.AddDays(-15), InspectorName: "孙涛", InspectorOrg: "市政排水管理处质检科",
				Result: acceptance.ResultRework, Score: 55, ResidualSludgeMm: 45,
				Issues:          "HDPE 管段错口未处理，残留淤积厚度 45 mm 超出 20 mm 的验收标准",
				Rectification:   "联系管网维修班组对错口段进行内衬修复，处理后重新清淤并复检",
				RectifyDeadline: ptrDate(today.AddDays(-10)), RectifiedAt: ptrDate(today.AddDays(-8)),
				Remark: "整改完成后需重新提交完工报验",
			},
		}
		if err := tx.Create(&acceptances).Error; err != nil {
			return err
		}

		logger.Info("演示数据初始化完成",
			"segments", len(segments),
			"tasks", len(tasks),
			"records", len(records),
			"acceptances", len(acceptances),
		)
		return nil
	})
}

// ptrDate 返回日期指针，便于构造可空字段。
func ptrDate(value date.Date) *date.Date {
	return &value
}

// stamp 把业务日期转成带具体时分的本地时间，用于 started_at / finished_at 等时间戳。
func stamp(day date.Date, hour int) *time.Time {
	moment := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, time.Local)
	return &moment
}
