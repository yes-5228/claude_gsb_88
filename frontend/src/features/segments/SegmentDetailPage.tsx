// 管段详情：基础档案 + 任务统计 + 最近任务 + 清淤履历。
import { Link, useNavigate, useParams } from 'react-router-dom';
import { toErrorMessage } from '../../api/client';
import { segmentApi } from '../../api/pipesegments';
import { ConfirmDialog } from '../../components/ConfirmDialog';
import { DataTable, type Column } from '../../components/DataTable';
import { InfoList } from '../../components/InfoList';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { StatCard } from '../../components/StatCard';
import { StatusTag } from '../../components/StatusTag';
import { StateBlock } from '../../components/StateBlock';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import type { SegmentHistoryItem, TaskRef } from '../../types/domain';
import { formatDate, formatDateTime, formatLength, formatNumber, formatVolume } from '../../utils/format';
import { useState } from 'react';

const taskColumns: Column<TaskRef>[] = [
  {
    key: 'code',
    title: '任务编号',
    width: '170px',
    render: (row) => (
      <>
        <Link className="cell-main" to={`/tasks/${row.id}`}>
          {row.code}
        </Link>
        <span className="cell-sub">{row.title}</span>
      </>
    )
  },
  { key: 'status', title: '状态', width: '100px', render: (row) => <StatusTag list="taskStatuses" value={row.status} /> },
  { key: 'priority', title: '优先级', width: '90px', render: (row) => <StatusTag list="taskPriorities" value={row.priority} /> },
  { key: 'teamName', title: '实施班组', width: '120px', render: (row) => row.teamName || '—' },
  {
    key: 'plan',
    title: '计划周期',
    width: '190px',
    render: (row) => `${formatDate(row.planStartDate)} ~ ${formatDate(row.planEndDate)}`
  },
  { key: 'recordCount', title: '记录数', width: '80px', align: 'right', render: (row) => formatNumber(row.recordCount, 0) },
  { key: 'sludge', title: '清淤量', width: '110px', align: 'right', render: (row) => formatVolume(row.sludgeVolumeM3) }
];

export function SegmentDetailPage() {
  const params = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const id = Number(params.id ?? '0');

  const detail = useAsync(
    () => (id > 0 ? segmentApi.detail(id) : Promise.reject(new Error('管段编号无效'))),
    [id]
  );
  const history = useAsync(
    () => (id > 0 ? segmentApi.history(id) : Promise.reject(new Error('管段编号无效'))),
    [id]
  );

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const segment = detail.data?.segment;
  const stats = detail.data?.taskStats;

  const handleDelete = async () => {
    if (!segment) {
      return;
    }
    setDeleting(true);
    try {
      await segmentApi.remove(segment.id);
      toast.success(`管段 ${segment.code} 已删除`);
      navigate('/segments');
    } catch (cause: unknown) {
      toast.error(toErrorMessage(cause));
      setConfirmOpen(false);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="page">
      <PageHeader
        title={segment ? `${segment.code} ${segment.name}` : '管段详情'}
        description="查看管段基础档案、关联清淤任务与历史清淤履历。"
        extra={segment ? <StatusTag list="segmentStatuses" value={segment.status} /> : null}
        actions={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => navigate('/segments')}>
              返回列表
            </button>
            <button
              type="button"
              className="btn btn-ghost"
              disabled={!segment}
              onClick={() => navigate(`/segments/${id}/edit`)}
            >
              编辑
            </button>
            <button type="button" className="btn btn-danger" disabled={!segment} onClick={() => setConfirmOpen(true)}>
              删除
            </button>
          </>
        }
      />

      <StateBlock loading={detail.loading} error={detail.error} onRetry={detail.reload} empty={!segment} emptyText="管段不存在">
        {segment ? (
          <>
            <SectionCard title="管段档案" subtitle={`创建于 ${formatDateTime(segment.createdAt)}`}>
              <InfoList
                items={[
                  { label: '管段编号', value: segment.code },
                  { label: '管段名称', value: segment.name },
                  { label: '所属片区', value: segment.district },
                  { label: '所在道路', value: segment.roadName || '—' },
                  { label: '管段类型', value: <StatusTag list="pipeTypes" value={segment.pipeType} /> },
                  { label: '管材', value: segment.material || '—' },
                  { label: '管径', value: `DN${segment.diameterMm}` },
                  { label: '管段长度', value: formatLength(segment.lengthM) },
                  { label: '埋深', value: `${formatNumber(segment.depthM, 2)} m` },
                  { label: '起始检查井', value: segment.startManhole || '—' },
                  { label: '终点检查井', value: segment.endManhole || '—' },
                  { label: '建设年份', value: segment.buildYear ? `${segment.buildYear} 年` : '—' },
                  { label: '权属单位', value: segment.ownerUnit || '—' },
                  { label: '最近清淤日期', value: formatDate(segment.lastCleanedAt) },
                  { label: '累计清淤次数', value: `${segment.cleanedTimes} 次` },
                  { label: '备注', value: segment.remark || '—', span: 3 }
                ]}
              />
            </SectionCard>

            <SectionCard title="任务概况" subtitle="该管段下清淤任务按状态统计">
              <div className="stat-grid">
                <StatCard label="任务总数" value={formatNumber(stats?.total ?? 0, 0)} tone="primary" />
                <StatCard label="待开工" value={formatNumber(stats?.pending ?? 0, 0)} />
                <StatCard label="清淤中" value={formatNumber(stats?.inProgress ?? 0, 0)} tone="warn" />
                <StatCard label="待验收" value={formatNumber(stats?.completed ?? 0, 0)} tone="warn" />
                <StatCard label="已验收" value={formatNumber(stats?.accepted ?? 0, 0)} tone="success" />
                <StatCard label="已取消" value={formatNumber(stats?.cancelled ?? 0, 0)} />
              </div>
            </SectionCard>

            <SectionCard title="最近任务" subtitle="按计划开始日期倒序展示最近 5 条任务">
              <div className="card-body-flush">
                <DataTable
                  columns={taskColumns}
                  rows={detail.data?.recentTasks ?? []}
                  rowKey={(row) => row.id}
                  emptyText="该管段暂无清淤任务"
                />
              </div>
            </SectionCard>

            <SectionCard title="清淤履历" subtitle="一次任务串起「计划 → 清淤记录 → 验收结论」">
              <StateBlock loading={history.loading} error={history.error} onRetry={history.reload}>
                {(history.data ?? []).length === 0 ? (
                  <p className="form-note">该管段还没有清淤履历。</p>
                ) : (
                  <ol className="timeline">
                    {(history.data ?? []).map((item: SegmentHistoryItem) => (
                      <li key={item.taskId} className="timeline-item">
                        <div className="timeline-title">
                          <Link to={`/tasks/${item.taskId}`}>{item.taskCode}</Link>
                          <span>{item.title}</span>
                          <StatusTag list="taskStatuses" value={item.status} />
                          <StatusTag list="taskPriorities" value={item.priority} />
                          {item.acceptanceResult ? (
                            <StatusTag list="acceptanceResults" value={item.acceptanceResult} />
                          ) : null}
                        </div>
                        <p className="timeline-meta">
                          计划 {formatDate(item.planStartDate)} ~ {formatDate(item.planEndDate)} · 班组{' '}
                          {item.teamName || '—'} · 清淤记录 {item.recordCount} 条 · 清淤量{' '}
                          {formatVolume(item.sludgeVolumeM3)} · 清淤长度 {formatLength(item.cleanedLengthM)}
                        </p>
                        <p className="timeline-meta">验收日期：{formatDate(item.acceptedAt)}</p>
                      </li>
                    ))}
                  </ol>
                )}
              </StateBlock>
            </SectionCard>
          </>
        ) : null}
      </StateBlock>

      <ConfirmDialog
        open={confirmOpen}
        title="删除管段"
        danger
        busy={deleting}
        confirmText="确认删除"
        message={<p>删除后不可恢复。若该管段已被清淤任务引用，后端会拒绝删除。</p>}
        onConfirm={handleDelete}
        onCancel={() => setConfirmOpen(false)}
      />
    </div>
  );
}
