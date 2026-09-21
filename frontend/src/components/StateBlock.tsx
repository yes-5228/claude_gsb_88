// 详情类页面的加载 / 失败 / 空数据占位，避免每个页面重复判断。
import type { ReactNode } from 'react';
import { EmptyState } from './EmptyState';

interface StateBlockProps {
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  empty?: boolean;
  emptyText?: string;
  children: ReactNode;
}

export function StateBlock({ loading, error, onRetry, empty, emptyText, children }: StateBlockProps) {
  if (loading) {
    return <div className="state-block">数据加载中…</div>;
  }
  if (error) {
    return (
      <div className="state-block state-block-error">
        <p>{error}</p>
        {onRetry ? (
          <button type="button" className="btn btn-ghost" onClick={onRetry}>
            重新加载
          </button>
        ) : null}
      </div>
    );
  }
  if (empty) {
    return (
      <div className="state-block">
        <EmptyState title={emptyText ?? '暂无数据'} />
      </div>
    );
  }
  return <>{children}</>;
}
