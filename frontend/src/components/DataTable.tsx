// 通用表格：内置加载中、失败重试与空数据三种状态。
import type { ReactNode } from 'react';
import { EmptyState } from './EmptyState';

export interface Column<T> {
  key: string;
  title: string;
  width?: string;
  align?: 'left' | 'center' | 'right';
  render: (row: T) => ReactNode;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string | number;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  emptyText?: string;
  emptyDescription?: string;
  onRowClick?: (row: T) => void;
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  loading,
  error,
  onRetry,
  emptyText,
  emptyDescription,
  onRowClick
}: DataTableProps<T>) {
  if (loading) {
    return <div className="table-state">数据加载中…</div>;
  }
  if (error) {
    return (
      <div className="table-state table-state-error">
        <p>{error}</p>
        {onRetry ? (
          <button type="button" className="btn btn-ghost" onClick={onRetry}>
            重新加载
          </button>
        ) : null}
      </div>
    );
  }
  if (rows.length === 0) {
    return (
      <EmptyState title={emptyText ?? '暂无数据'} description={emptyDescription} />
    );
  }

  return (
    <div className="table-wrap">
      <table className="data-table">
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                style={{ width: column.width, textAlign: column.align ?? 'left' }}
              >
                {column.title}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={rowKey(row)}
              className={onRowClick ? 'row-clickable' : undefined}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
            >
              {columns.map((column) => (
                <td key={column.key} style={{ textAlign: column.align ?? 'left' }}>
                  {column.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
