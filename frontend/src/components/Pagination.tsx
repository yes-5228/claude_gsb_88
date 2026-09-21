// 分页条。
interface PaginationProps {
  total: number;
  page: number;
  pageSize: number;
  onChange: (page: number) => void;
}

export function Pagination({ total, page, pageSize, onChange }: PaginationProps) {
  if (total <= 0) {
    return null;
  }
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const from = (page - 1) * pageSize + 1;
  const to = Math.min(total, page * pageSize);

  return (
    <div className="pagination">
      <span className="pagination-info">
        共 {total} 条，当前第 {from}-{to} 条
      </span>
      <div className="pagination-actions">
        <button type="button" className="btn btn-ghost" disabled={page <= 1} onClick={() => onChange(page - 1)}>
          上一页
        </button>
        <span className="pagination-page">
          {page} / {pageCount}
        </span>
        <button
          type="button"
          className="btn btn-ghost"
          disabled={page >= pageCount}
          onClick={() => onChange(page + 1)}
        >
          下一页
        </button>
      </div>
    </div>
  );
}
