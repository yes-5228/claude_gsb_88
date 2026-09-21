// 管段台账列表：按片区、类型、状态与关键字检索，支持新增 / 编辑 / 删除。
import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toErrorMessage } from '../../api/client';
import { segmentApi } from '../../api/pipesegments';
import { ConfirmDialog } from '../../components/ConfirmDialog';
import { DataTable, type Column } from '../../components/DataTable';
import { PageHeader } from '../../components/PageHeader';
import { Pagination } from '../../components/Pagination';
import { SectionCard } from '../../components/SectionCard';
import { StatusTag } from '../../components/StatusTag';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { PipeSegment } from '../../types/domain';
import { formatDate, formatLength, formatNumber } from '../../utils/format';

const PAGE_SIZE = 10;

export function SegmentListPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();
  const [params, setParams] = useSearchParams();

  const keyword = params.get('keyword') ?? '';
  const district = params.get('district') ?? '';
  const pipeType = params.get('pipeType') ?? '';
  const status = params.get('status') ?? '';
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1);

  const [keywordInput, setKeywordInput] = useState(keyword);
  useEffect(() => {
    setKeywordInput(keyword);
  }, [keyword]);

  const list = useAsync(
    () => segmentApi.list({ keyword, district, pipeType, status, page, pageSize: PAGE_SIZE }),
    [keyword, district, pipeType, status, page]
  );
  const options = useAsync(() => segmentApi.options(), []);

  const [pendingDelete, setPendingDelete] = useState<PipeSegment | null>(null);
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
      await segmentApi.remove(pendingDelete.id);
      toast.success(`管段 ${pendingDelete.code} 已删除`);
      setPendingDelete(null);
      list.reload();
      options.reload();
    } catch (cause: unknown) {
      toast.error(toErrorMessage(cause));
    } finally {
      setDeleting(false);
    }
  };

  const columns: Column<PipeSegment>[] = [
    {
      key: 'code',
      title: '管段编号',
      width: '170px',
      render: (row) => (
        <>
          <Link className="cell-main" to={`/segments/${row.id}`}>
            {row.code}
          </Link>
          <span className="cell-sub">{row.name}</span>
        </>
      )
    },
    {
      key: 'district',
      title: '片区 / 道路',
      render: (row) => (
        <>
          <span>{row.district}</span>
          <span className="cell-sub">{row.roadName || '—'}</span>
        </>
      )
    },
    {
      key: 'pipeType',
      title: '类型 / 管材',
      render: (row) => (
        <>
          <StatusTag list="pipeTypes" value={row.pipeType} />
          <span className="cell-sub">
            {row.material || '—'} · DN{row.diameterMm}
          </span>
        </>
      )
    },
    {
      key: 'lengthM',
      title: '长度 / 埋深',
      align: 'right',
      render: (row) => (
        <>
          <span className="cell-num">{formatLength(row.lengthM)}</span>
          <span className="cell-sub">埋深 {formatNumber(row.depthM, 2)} m</span>
        </>
      )
    },
    {
      key: 'status',
      title: '运行状态',
      width: '110px',
      render: (row) => <StatusTag list="segmentStatuses" value={row.status} />
    },
    {
      key: 'lastCleanedAt',
      title: '最近清淤',
      width: '130px',
      align: 'right',
      render: (row) => (
        <>
          <span>{formatDate(row.lastCleanedAt)}</span>
          <span className="cell-sub">累计 {row.cleanedTimes} 次</span>
        </>
      )
    },
    {
      key: 'actions',
      title: '操作',
      width: '150px',
      render: (row) => (
        <div className="row-actions">
          <Link className="link" to={`/segments/${row.id}`}>
            详情
          </Link>
          <Link className="link" to={`/segments/${row.id}/edit`}>
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
        title="管段台账"
        description="维护排水管网管段基础档案，记录清淤次数与最近清淤时间，作为任务与记录的关联主体。"
        actions={
          <button type="button" className="btn btn-primary" onClick={() => navigate('/segments/new')}>
            新增管段
          </button>
        }
      />

      <SectionCard title="管段清单" subtitle={`共 ${list.data?.total ?? 0} 条记录`}>
        <div className="card-body-flush">
          <div className="filter-bar">
            <div className="filter-item" style={{ minWidth: 220 }}>
              <span className="filter-label">关键字</span>
              <input
                className="input"
                placeholder="编号 / 名称 / 道路 / 检查井"
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
              <span className="filter-label">所属片区</span>
              <select
                className="select"
                value={district}
                onChange={(event) => applyFilter({ district: event.target.value })}
              >
                <option value="">全部片区</option>
                {(options.data?.districts ?? []).map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">管段类型</span>
              <select
                className="select"
                value={pipeType}
                onChange={(event) => applyFilter({ pipeType: event.target.value })}
              >
                <option value="">全部类型</option>
                {(enums?.pipeTypes ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">运行状态</span>
              <select
                className="select"
                value={status}
                onChange={(event) => applyFilter({ status: event.target.value })}
              >
                <option value="">全部状态</option>
                {(enums?.segmentStatuses ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
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
            emptyText="未找到符合条件的管段"
            emptyDescription="可以调整筛选条件，或先新增管段档案。"
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
        title="删除管段"
        danger
        busy={deleting}
        confirmText="确认删除"
        message={
          <>
            <p>
              即将删除管段 <strong>{pendingDelete?.code}</strong>（{pendingDelete?.name}）。
            </p>
            <p>已关联清淤任务的管段不允许删除，请先处理关联任务。</p>
          </>
        }
        onConfirm={handleDelete}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
