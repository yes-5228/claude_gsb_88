// 运行看板：跨模块汇总管段、任务、清淤与验收数据。
import { Link, useNavigate } from 'react-router-dom';
import { dashboardApi } from '../../api/dashboard';
import { DataTable, type Column } from '../../components/DataTable';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { StatCard } from '../../components/StatCard';
import { StateBlock } from '../../components/StateBlock';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { DistrictStat, PendingAcceptanceItem, RecentRecordItem } from '../../types/domain';
import { formatDate, formatLength, formatNumber, formatPercent, formatVolume } from '../../utils/format';
import { optionLabel } from '../../utils/options';

interface BarItem {
  label: string;
  value: number;
  hint?: string;
}

function BarList({ items, emptyText }: { items: BarItem[]; emptyText: string }) {
  if (items.length === 0) {
    return <p className="form-note">{emptyText}</p>;
  }
  const max = Math.max(1, ...items.map((item) => item.value));
  return (
    <div className="bar-list">
      {items.map((item) => (
        <div key={item.label} className="bar-row">
          <span>{item.label}</span>
          <div className="bar-track">
            <div className="bar-fill" style={{ width: `${(item.value / max) * 100}%` }} />
          </div>
          <span className="bar-value">{item.hint ?? formatNumber(item.value, 0)}</span>
        </div>
      ))}
    </div>
  );
}

const pendingColumns: Column<PendingAcceptanceItem>[] = [
  { key: 'code', title: '任务编号', width: '140px', render: (row) => <span className="cell-main">{row.code}</span> },
  {
    key: 'title',
    title: '任务与管段',
    render: (row) => (
      <>
        <span>{row.title}</span>
        <span className="cell-sub">
          {row.segmentCode} · {row.segmentName}
        </span>
      </>
    )
  },
  { key: 'teamName', title: '实施班组', width: '120px', render: (row) => row.teamName || '—' },
  { key: 'finishedAt', title: '完工时间', width: '110px', render: (row) => formatDate(row.finishedAt) },
  {
    key: 'overdueDays',
    title: '超期',
    width: '90px',
    align: 'right',
    render: (row) =>
      row.overdueDays > 0 ? <span className="tag tag-danger">{row.overdueDays} 天</span> : <span className="tag tag-muted">正常</span>
  },
  {
    key: 'action',
    title: '操作',
    width: '90px',
    render: (row) => (
      <Link className="link" to={`/tasks/${row.taskId}`}>
        查看任务
      </Link>
    )
  }
];

const recentColumns: Column<RecentRecordItem>[] = [
  { key: 'code', title: '记录编号', width: '140px', render: (row) => <span className="cell-main">{row.code}</span> },
  { key: 'cleanedAt', title: '清淤日期', width: '110px', render: (row) => formatDate(row.cleanedAt) },
  {
    key: 'task',
    title: '所属任务与管段',
    render: (row) => (
      <>
        <span>{row.taskTitle}</span>
        <span className="cell-sub">
          {row.segmentCode} · {row.segmentName}
        </span>
      </>
    )
  },
  { key: 'lengthM', title: '清淤长度', width: '110px', align: 'right', render: (row) => formatLength(row.lengthM) },
  { key: 'sludgeVolumeM3', title: '清淤量', width: '110px', align: 'right', render: (row) => formatVolume(row.sludgeVolumeM3) }
];

export function DashboardPage() {
  const navigate = useNavigate();
  const { enums } = useMeta();
  const overview = useAsync(() => dashboardApi.overview(), []);
  const districts = useAsync(() => dashboardApi.districtStats(), []);
  const pending = useAsync(() => dashboardApi.pendingAcceptance(6), []);
  const recent = useAsync(() => dashboardApi.recentRecords(6), []);

  const data = overview.data;
  const taskStatusBars: BarItem[] = enums
    ? enums.taskStatuses.map((status) => ({
        label: status.label,
        value: data?.taskByStatus?.[status.value] ?? 0
      }))
    : [];
  const segmentStatusBars: BarItem[] = enums
    ? enums.segmentStatuses.map((status) => ({
        label: status.label,
        value: data?.segmentByStatus?.[status.value] ?? 0
      }))
    : [];

  const districtColumns: Column<DistrictStat>[] = [
    { key: 'district', title: '片区', render: (row) => <span className="cell-main">{row.district}</span> },
    { key: 'segmentCount', title: '管段', align: 'right', render: (row) => formatNumber(row.segmentCount, 0) },
    { key: 'segmentLengthM', title: '总长', align: 'right', render: (row) => formatLength(row.segmentLengthM) },
    { key: 'taskCount', title: '任务', align: 'right', render: (row) => formatNumber(row.taskCount, 0) },
    { key: 'sludgeVolumeM3', title: '清淤量', align: 'right', render: (row) => formatVolume(row.sludgeVolumeM3) },
    { key: 'lastCleanedAt', title: '最近清淤', align: 'right', render: (row) => formatDate(row.lastCleanedAt) }
  ];

  return (
    <div className="page">
      <PageHeader
        title="运行看板"
        description="汇总管网台账、清淤任务、清淤记录与验收结论，用于掌握整体进度与待办事项。"
        actions={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => navigate('/segments/new')}>
              新增管段
            </button>
            <button type="button" className="btn btn-primary" onClick={() => navigate('/tasks/new')}>
              登记清淤任务
            </button>
          </>
        }
      />

      <StateBlock loading={overview.loading} error={overview.error} onRetry={overview.reload}>
        <div className="stat-grid">
          <StatCard
            label="管段总数"
            value={formatNumber(data?.segmentTotal ?? 0, 0)}
            hint={`总长度 ${formatLength(data?.segmentTotalLengthM ?? 0)}`}
            tone="primary"
            onClick={() => navigate('/segments')}
          />
          <StatCard
            label="未清淤管段"
            value={formatNumber(data?.uncleanedSegmentCount ?? 0, 0)}
            hint="尚未产生已验收清淤记录的管段"
            tone={data && data.uncleanedSegmentCount > 0 ? 'warn' : 'success'}
          />
          <StatCard
            label="清淤任务"
            value={formatNumber(data?.taskTotal ?? 0, 0)}
            hint={`待开工 ${data?.taskByStatus?.pending ?? 0} · 清淤中 ${data?.taskByStatus?.in_progress ?? 0}`}
            onClick={() => navigate('/tasks')}
          />
          <StatCard
            label="待验收任务"
            value={formatNumber(data?.pendingAcceptanceCount ?? 0, 0)}
            hint={`其中超期 ${data?.taskOverdue ?? 0} 项`}
            tone={data && data.pendingAcceptanceCount > 0 ? 'warn' : 'default'}
            onClick={() => navigate('/tasks?status=completed')}
          />
          <StatCard
            label="清淤记录"
            value={formatNumber(data?.recordTotal ?? 0, 0)}
            hint={`累计清淤长度 ${formatLength(data?.cleanedLengthM ?? 0)}`}
            onClick={() => navigate('/records')}
          />
          <StatCard label="累计清淤量" value={formatVolume(data?.sludgeTotalM3 ?? 0)} hint="按清淤记录汇总" />
          <StatCard label="本月清淤量" value={formatVolume(data?.sludgeThisMonthM3 ?? 0)} hint="当月清淤日期口径" />
          <StatCard
            label="验收合格率"
            value={formatPercent(data?.acceptancePassRate ?? 0)}
            hint={`合格 ${data?.acceptancePassCount ?? 0} / 共 ${data?.acceptanceTotal ?? 0} 次`}
            tone="success"
            onClick={() => navigate('/acceptances')}
          />
        </div>
      </StateBlock>

      <div className="panel-grid panel-grid-wide">
        <SectionCard
          title="待验收任务"
          subtitle="已完工报验但尚无验收结论的任务"
          extra={<Link className="link" to="/tasks?status=completed">全部待验收</Link>}
        >
          <div className="card-body-flush">
            <DataTable
              columns={pendingColumns}
              rows={pending.data ?? []}
              rowKey={(row) => row.taskId}
              loading={pending.loading}
              error={pending.error}
              onRetry={pending.reload}
              emptyText="暂无待验收任务"
            />
          </div>
        </SectionCard>

        <SectionCard title="任务状态分布" subtitle="按清淤任务状态统计">
          <BarList items={taskStatusBars} emptyText="暂无任务数据" />
          <div style={{ height: 16 }} />
          <p className="form-note">管段运行状态</p>
          <div style={{ height: 8 }} />
          <BarList items={segmentStatusBars} emptyText="暂无管段数据" />
        </SectionCard>
      </div>

      <div className="panel-grid panel-grid-wide">
        <SectionCard
          title="最近清淤记录"
          subtitle="按清淤日期倒序展示最新录入结果"
          extra={<Link className="link" to="/records">查看全部记录</Link>}
        >
          <div className="card-body-flush">
            <DataTable
              columns={recentColumns}
              rows={recent.data ?? []}
              rowKey={(row) => row.recordId}
              loading={recent.loading}
              error={recent.error}
              onRetry={recent.reload}
              emptyText="暂无清淤记录"
            />
          </div>
        </SectionCard>

        <SectionCard title="分片区统计" subtitle="按片区汇总台账与清淤成果">
          <div className="card-body-flush">
            <DataTable
              columns={districtColumns}
              rows={districts.data ?? []}
              rowKey={(row) => row.district}
              loading={districts.loading}
              error={districts.error}
              onRetry={districts.reload}
              emptyText="暂无片区数据"
            />
          </div>
        </SectionCard>
      </div>

      <p className="form-note">
        说明：验收合格率 = 合格验收次数 / 验收总次数；未清淤管段指尚无「验收合格」记录的管段，
        与管段台账中的最近清淤日期口径一致。字典标签取自后端 {optionLabel(enums?.acceptanceResults, 'pass')} 等统一枚举。
      </p>
    </div>
  );
}
