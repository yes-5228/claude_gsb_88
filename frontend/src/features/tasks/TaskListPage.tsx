// 清淤任务列表：按状态、片区、优先级、来源与关键字检索。
import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toErrorMessage } from '../../api/client';
import { taskApi } from '../../api/tasks';
import { ConfirmDialog } from '../../components/ConfirmDialog';
import { DataTable, type Column } from '../../components/DataTable';
import { PageHeader } from '../../components/PageHeader';
import { Pagination } from '../../components/Pagination';
import { SectionCard } from '../../components/SectionCard';
import { StatusTag } from '../../components/StatusTag';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { TaskListItem } from '../../types/domain';
import { formatDate, formatNumber, formatVolume } from '../../utils/format';

const PAGE_SIZE = 10;

export function TaskListPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();
  const [params, setParams] = useSearchParams();

  const keyword = params.get('keyword') ?? '';
  const status = params.get('status') ?? '';
  const district = params.get('district') ?? '';
  const priority = params.get('priority') ?? '';
  const source = params.get('source') ?? '';
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1);

  const [keywordInput, setKeywordInput] = useState(keyword);
  useEffect(() => {
    setKeywordInput(keyword);
  }, [keyword]);

  const list = useAsync(
    () => taskApi.list({ keyword, status, district, priority, source, page, pageSize: PAGE_SIZE }),
    [keyword, status, district, priority, source, page]
  );

  const [pendingDelete, setPendingDelete] = useState<TaskListItem | null>(null);
  const [deleting, setDeleting] = useState(false);

  const applyFilter = (patch: Record<string, string>) => {
    const next = new URLSearchParams(params);
    Object.entries(patch).forEach(([key, value]) => {
      if (value) {
        next.set(key, value);
      } else {
        next.delete(key);
      }
    });
    next.set('page', '1');
    setParams(next);
  };

  const goPage = (nextPage: number) => {
    const next = new URLSearchParams(params);
    next.set('page', String(nextPage));
    setParams(next);
  };

  const handleDelete = async () => {
    if (!pendingDelete) {
      return;
    }
    setDeleting(true);
    try {
      await taskApi.remove(pendingDelete.id);
      toast.success(`任务 ${pendingDelete.code} 已删除`);
      setPendingDelete(null);
      list.reload();
    } catch (cause: unknown) {
      toast.error(toErrorMessage(cause));
    } finally {
      setDeleting(false);
    }
  };

  const columns: Column<TaskListItem>[] = [
    {
      key: 'code',
      title: '任务编号',
      width: '180px',
      render: (row) => (
        <>
          <Link className="cell-main" to={`/tasks/${row.id}`}>
            {row.code}
          </Link>
          <span className="cell-sub">{row.title}</span>
        </>
      )
    },
    {
      key: 'segment',
      title: '关联管段',
      width: '170px',
      render: (row) => (
        <>
          <span>{row.segment?.code ?? '—'}</span>
          <span className="cell-sub">
            {row.segment ? `${row.segment.district} · ${row.segment.name}` : '管段已删除'}
          </span>
        </>
      )
    },
    { key: 'status', title: '状态', width: '100px', render: (row) => <StatusTag list="taskStatuses" value={row.status} /> },
    {
      key: 'priority',
      title: '优先级',
      width: '90px',
      render: (row) => <StatusTag list="taskPriorities" value={row.priority} />
    },
    { key: 'source', title: '来源', width: '110px', render: (row) => <StatusTag list="taskSources" value={row.source} /> },
    {
      key: 'team',
      title: '实施班组',
      width: '140px',
      render: (row) => (
        <>
          <span>{row.teamName || '—'}</span>
          <span className="cell-sub">{row.leaderName || '未指定负责人'}</span>
        </>
      )
    },
    {
      key: 'plan',
      title: '计划周期',
      width: '190px',
      render: (row) => `${formatDate(row.planStartDate)} ~ ${formatDate(row.planEndDate)}`
    },
    {
      key: 'totals',
      title: '清淤汇总',
      width: '130px',
      align: 'right',
      render: (row) => (
        <>
          <span className="cell-num">{formatVolume(row.recordTotals?.sludgeVolumeM3 ?? 0)}</span>
          <span className="cell-sub">记录 {formatNumber(row.recordTotals?.recordCount ?? 0, 0)} 条</span>
        </>
      )
    },
    {
      key: 'actions',
      title: '操作',
      width: '150px',
      render: (row) => (
        <div className="row-actions">
          <Link className="link" to={`/tasks/${row.id}`}>
            详情
          </Link>
          <Link className="link" to={`/tasks/${row.id}/edit`}>
            编辑
          </Link>
          <button type="button" className="btn-link" onClick={() => setPendingDelete(row)}>
            删除
          </button>
        </div>
      )
    }
  ];

  return (
    <div className="page">
      <PageHeader
        title="清淤任务"
        description="登记年度计划、巡查发现与投诉举报产生的清淤任务，驱动清淤记录录入与验收流程。"
        actions={
          <button type="button" className="btn btn-primary" onClick={() => navigate('/tasks/new')}>
            登记任务
          </button>
        }
      />

      <SectionCard title="任务清单" subtitle={`共 ${list.data?.total ?? 0} 条记录`}>
        <div className="card-body-flush">
          <div className="filter-bar">
            <div className="filter-item" style={{ minWidth: 220 }}>
              <span className="filter-label">关键字</span>
              <input
                className="input"
                placeholder="任务编号 / 标题 / 班组"
                value={keywordInput}
                onChange={(event) => setKeywordInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    applyFilter({ keyword: keywordInput });
                  }
                }}
              />
            </div>
            <div className="filter-item">
              <span className="filter-label">任务状态</span>
              <select className="select" value={status} onChange={(event) => applyFilter({ status: event.target.value })}>
                <option value="">全部状态</option>
                {(enums?.taskStatuses ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">优先级</span>
              <select
                className="select"
                value={priority}
                onChange={(event) => applyFilter({ priority: event.target.value })}
              >
                <option value="">全部优先级</option>
                {(enums?.taskPriorities ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">任务来源</span>
              <select className="select" value={source} onChange={(event) => applyFilter({ source: event.target.value })}>
                <option value="">全部来源</option>
                {(enums?.taskSources ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">所属片区</span>
              <input
                className="input"
                placeholder="精确匹配片区"
                value={district}
                onChange={(event) => applyFilter({ district: event.target.value })}
              />
            </div>
            <div className="filter-actions">
              <button type="button" className="btn btn-ghost" onClick={() => setParams(new URLSearchParams())}>
                重置
              </button>
              <button type="button" className="btn btn-primary" onClick={() => applyFilter({ keyword: keywordInput })}>
                查询
              </button>
            </div>
          </div>

          <DataTable
            columns={columns}
            rows={list.data?.list ?? []}
            rowKey={(row) => row.id}
            loading={list.loading}
            error={list.error}
            onRetry={list.reload}
            emptyText="未找到符合条件的清淤任务"
            emptyDescription="可以先登记任务，再录入清淤记录。"
          />
          <Pagination
            total={list.data?.total ?? 0}
            page={page}
            pageSize={list.data?.pageSize ?? PAGE_SIZE}
            onChange={goPage}
          />
        </div>
      </SectionCard>

      <ConfirmDialog
        open={pendingDelete !== null}
        title="删除清淤任务"
        danger
        busy={deleting}
        confirmText="确认删除"
        message={
          <>
            <p>
              即将删除任务 <strong>{pendingDelete?.code}</strong>（{pendingDelete?.title}）。
            </p>
            <p>已录入清淤记录或已产生验收记录的任务不允许删除，以保证台账可追溯。</p>
          </>
        }
        onConfirm={handleDelete}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
