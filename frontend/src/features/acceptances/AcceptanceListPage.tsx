// 验收记录列表：按结论、验收人、验收日期与整改状态检索。
import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { acceptanceApi } from '../../api/acceptances';
import { DataTable, type Column } from '../../components/DataTable';
import { PageHeader } from '../../components/PageHeader';
import { Pagination } from '../../components/Pagination';
import { SectionCard } from '../../components/SectionCard';
import { StatusTag } from '../../components/StatusTag';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { AcceptanceListItem } from '../../types/domain';
import { formatDate, formatNumber } from '../../utils/format';

const PAGE_SIZE = 10;

export function AcceptanceListPage() {
  const navigate = useNavigate();
  const { enums } = useMeta();
  const [params, setParams] = useSearchParams();

  const keyword = params.get('keyword') ?? '';
  const result = params.get('result') ?? '';
  const inspectorName = params.get('inspectorName') ?? '';
  const dateFrom = params.get('dateFrom') ?? '';
  const dateTo = params.get('dateTo') ?? '';
  const pendingRectify = params.get('pendingRectify') === 'true';
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1);

  const [keywordInput, setKeywordInput] = useState(keyword);
  useEffect(() => {
    setKeywordInput(keyword);
  }, [keyword]);

  const list = useAsync(
    () =>
      acceptanceApi.list({
        keyword,
        result,
        inspectorName,
        dateFrom,
        dateTo,
        pendingRectify: pendingRectify ? true : undefined,
        page,
        pageSize: PAGE_SIZE
      }),
    [keyword, result, inspectorName, dateFrom, dateTo, pendingRectify, page]
  );

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

  const columns: Column<AcceptanceListItem>[] = [
    {
      key: 'code',
      title: '验收编号',
      width: '180px',
      render: (row) => (
        <>
          <Link className="cell-main" to={`/acceptances/${row.id}`}>
            {row.code}
          </Link>
          <span className="cell-sub">验收日期 {formatDate(row.acceptedAt)}</span>
        </>
      )
    },
    {
      key: 'task',
      title: '所属任务 / 管段',
      width: '230px',
      render: (row) => (
        <>
          <span>{row.task?.title ?? '—'}</span>
          <span className="cell-sub">
            {row.task ? `${row.task.code} · ${row.task.segmentCode} ${row.task.segmentName}` : '任务已删除'}
          </span>
        </>
      )
    },
    {
      key: 'result',
      title: '验收结论',
      width: '100px',
      render: (row) => <StatusTag list="acceptanceResults" value={row.result} />
    },
    {
      key: 'score',
      title: '评分 / 残留淤积',
      width: '140px',
      align: 'right',
      render: (row) => (
        <>
          <span className="cell-num">{row.score} 分</span>
          <span className="cell-sub">残留 {formatNumber(row.residualSludgeMm, 1)} mm</span>
        </>
      )
    },
    {
      key: 'inspector',
      title: '验收人 / 单位',
      width: '150px',
      render: (row) => (
        <>
          <span>{row.inspectorName}</span>
          <span className="cell-sub">{row.inspectorOrg || '—'}</span>
        </>
      )
    },
    {
      key: 'rectify',
      title: '整改情况',
      width: '170px',
      render: (row) => {
        if (row.result !== 'rework') {
          return <span className="tag tag-muted">无需整改</span>;
        }
        if (row.rectifiedAt) {
          return (
            <>
              <span className="tag tag-success">已完成整改</span>
              <span className="cell-sub">{formatDate(row.rectifiedAt)}</span>
            </>
          );
        }
        return (
          <>
            <span className="tag tag-danger">待整改</span>
            <span className="cell-sub">期限 {formatDate(row.rectifyDeadline)}</span>
          </>
        );
      }
    },
    {
      key: 'actions',
      title: '操作',
      width: '150px',
      render: (row) => (
        <div className="row-actions">
          <Link className="link" to={`/acceptances/${row.id}`}>
            详情
          </Link>
          <Link className="link" to={`/tasks/${row.taskId}`}>
            任务
          </Link>
        </div>
      )
    }
  ];

  return (
    <div className="page">
      <PageHeader
        title="验收记录"
        description="对完工报验的清淤任务登记验收结论；验收合格后任务归档且管段清淤次数自动累计，需整改的任务回到清淤中。"
        actions={
          <button type="button" className="btn btn-primary" onClick={() => navigate('/acceptances/new')}>
            登记验收
          </button>
        }
      />

      <SectionCard title="验收清单" subtitle={`共 ${list.data?.total ?? 0} 条记录`}>
        <div className="card-body-flush">
          <div className="filter-bar">
            <div className="filter-item" style={{ minWidth: 220 }}>
              <span className="filter-label">关键字</span>
              <input
                className="input"
                placeholder="验收编号 / 验收人 / 单位"
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
              <span className="filter-label">验收结论</span>
              <select className="select" value={result} onChange={(event) => applyFilter({ result: event.target.value })}>
                <option value="">全部结论</option>
                {(enums?.acceptanceResults ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">验收人</span>
              <input
                className="input"
                value={inspectorName}
                onChange={(event) => applyFilter({ inspectorName: event.target.value })}
              />
            </div>
            <div className="filter-item">
              <span className="filter-label">验收日期起</span>
              <input
                className="input"
                type="date"
                value={dateFrom}
                onChange={(event) => applyFilter({ dateFrom: event.target.value })}
              />
            </div>
            <div className="filter-item">
              <span className="filter-label">验收日期止</span>
              <input
                className="input"
                type="date"
                value={dateTo}
                onChange={(event) => applyFilter({ dateTo: event.target.value })}
              />
            </div>
            <div className="filter-item">
              <span className="filter-label">整改状态</span>
              <select
                className="select"
                value={pendingRectify ? 'true' : ''}
                onChange={(event) => applyFilter({ pendingRectify: event.target.value })}
              >
                <option value="">全部</option>
                <option value="true">仅看待整改</option>
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
            emptyText="未找到符合条件的验收记录"
            emptyDescription="只有「待验收」的任务可以登记验收结论。"
          />
          <Pagination
            total={list.data?.total ?? 0}
            page={page}
            pageSize={list.data?.pageSize ?? PAGE_SIZE}
            onChange={goPage}
          />
        </div>
      </SectionCard>
    </div>
  );
}
