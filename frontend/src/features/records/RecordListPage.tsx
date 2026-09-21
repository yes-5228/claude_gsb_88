// 清淤记录列表：按任务、清淤方式、天气与清淤日期区间检索。
import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toErrorMessage } from '../../api/client';
import { recordApi } from '../../api/records';
import { ConfirmDialog } from '../../components/ConfirmDialog';
import { DataTable, type Column } from '../../components/DataTable';
import { PageHeader } from '../../components/PageHeader';
import { Pagination } from '../../components/Pagination';
import { SectionCard } from '../../components/SectionCard';
import { StatusTag } from '../../components/StatusTag';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { RecordListItem } from '../../types/domain';
import { formatDate, formatLength, formatNumber, formatVolume } from '../../utils/format';

const PAGE_SIZE = 10;

export function RecordListPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();
  const [params, setParams] = useSearchParams();

  const keyword = params.get('keyword') ?? '';
  const taskId = Number(params.get('taskId') ?? '0') || 0;
  const method = params.get('method') ?? '';
  const weather = params.get('weather') ?? '';
  const dateFrom = params.get('dateFrom') ?? '';
  const dateTo = params.get('dateTo') ?? '';
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1);

  const [keywordInput, setKeywordInput] = useState(keyword);
  useEffect(() => {
    setKeywordInput(keyword);
  }, [keyword]);

  const list = useAsync(
    () =>
      recordApi.list({
        keyword,
        taskId: taskId || undefined,
        method,
        weather,
        dateFrom,
        dateTo,
        page,
        pageSize: PAGE_SIZE
      }),
    [keyword, taskId, method, weather, dateFrom, dateTo, page]
  );

  const [pendingDelete, setPendingDelete] = useState<RecordListItem | null>(null);
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

  const clearTaskFilter = () => {
    const next = new URLSearchParams(params);
    next.delete('taskId');
    next.set('page', '1');
    setParams(next);
  };

  const handleDelete = async () => {
    if (!pendingDelete) {
      return;
    }
    setDeleting(true);
    try {
      await recordApi.remove(pendingDelete.id);
      toast.success(`清淤记录 ${pendingDelete.code} 已删除`);
      setPendingDelete(null);
      list.reload();
    } catch (cause: unknown) {
      toast.error(toErrorMessage(cause));
    } finally {
      setDeleting(false);
    }
  };

  const columns: Column<RecordListItem>[] = [
    {
      key: 'code',
      title: '记录编号',
      width: '180px',
      render: (row) => (
        <>
          <Link className="cell-main" to={`/records/${row.id}`}>
            {row.code}
          </Link>
          <span className="cell-sub">记录人 {row.recorderName || '—'}</span>
        </>
      )
    },
    { key: 'cleanedAt', title: '清淤日期', width: '110px', render: (row) => formatDate(row.cleanedAt) },
    {
      key: 'task',
      title: '所属任务 / 管段',
      width: '220px',
      render: (row) => (
        <>
          <span>{row.task?.code ?? '—'}</span>
          <span className="cell-sub">
            {row.task ? `${row.task.segmentCode} · ${row.task.segmentName}` : '任务已删除'}
          </span>
        </>
      )
    },
    {
      key: 'method',
      title: '清淤方式',
      width: '120px',
      render: (row) => <StatusTag list="cleaningMethods" value={row.method} />
    },
    { key: 'weather', title: '天气', width: '90px', render: (row) => <StatusTag list="weathers" value={row.weather} /> },
    { key: 'lengthM', title: '清淤长度', width: '110px', align: 'right', render: (row) => formatLength(row.lengthM) },
    {
      key: 'sludgeVolumeM3',
      title: '清淤量 / 用水量',
      width: '140px',
      align: 'right',
      render: (row) => (
        <>
          <span className="cell-num">{formatVolume(row.sludgeVolumeM3)}</span>
          <span className="cell-sub">用水 {formatVolume(row.waterVolumeM3)}</span>
        </>
      )
    },
    {
      key: 'personnelCount',
      title: '作业人数',
      width: '90px',
      align: 'right',
      render: (row) => formatNumber(row.personnelCount, 0)
    },
    {
      key: 'actions',
      title: '操作',
      width: '150px',
      render: (row) => (
        <div className="row-actions">
          <Link className="link" to={`/records/${row.id}`}>
            详情
          </Link>
          <Link className="link" to={`/records/${row.id}/edit`}>
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
        title="清淤记录"
        description="按次录入现场清淤数据（清淤长度、清淤量、用水量、作业人数与安全措施），首次录入会自动推进任务状态。"
        actions={
          <button type="button" className="btn btn-primary" onClick={() => navigate('/records/new')}>
            录入清淤记录
          </button>
        }
      />

      <SectionCard
        title="记录清单"
        subtitle={`共 ${list.data?.total ?? 0} 条记录`}
        extra={
          taskId ? (
            <button type="button" className="btn btn-ghost btn-sm" onClick={clearTaskFilter}>
              已按任务 #{taskId} 筛选，点击清除
            </button>
          ) : null
        }
      >
        <div className="card-body-flush">
          <div className="filter-bar">
            <div className="filter-item" style={{ minWidth: 220 }}>
              <span className="filter-label">关键字</span>
              <input
                className="input"
                placeholder="记录编号 / 记录人 / 设备"
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
              <span className="filter-label">清淤方式</span>
              <select className="select" value={method} onChange={(event) => applyFilter({ method: event.target.value })}>
                <option value="">全部方式</option>
                {(enums?.cleaningMethods ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">天气</span>
              <select className="select" value={weather} onChange={(event) => applyFilter({ weather: event.target.value })}>
                <option value="">全部天气</option>
                {(enums?.weathers ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">清淤日期起</span>
              <input
                className="input"
                type="date"
                value={dateFrom}
                onChange={(event) => applyFilter({ dateFrom: event.target.value })}
              />
            </div>
            <div className="filter-item">
              <span className="filter-label">清淤日期止</span>
              <input
                className="input"
                type="date"
                value={dateTo}
                onChange={(event) => applyFilter({ dateTo: event.target.value })}
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
            emptyText="未找到符合条件的清淤记录"
            emptyDescription="清淤记录必须挂在清淤任务下，请先登记任务。"
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
        title="删除清淤记录"
        danger
        busy={deleting}
        confirmText="确认删除"
        message={
          <>
            <p>
              即将删除清淤记录 <strong>{pendingDelete?.code}</strong>。
            </p>
            <p>任务已完工报验、或该记录已被验收记录引用时，后端会拒绝删除。</p>
          </>
        }
        onConfirm={handleDelete}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
