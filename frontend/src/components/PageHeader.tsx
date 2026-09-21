// 页面标题区：标题 + 说明 + 右侧操作按钮，各页面统一使用。
import type { ReactNode } from 'react';

interface PageHeaderProps {
  title: string;
  description?: string;
  extra?: ReactNode;
  actions?: ReactNode;
}

export function PageHeader({ title, description, extra, actions }: PageHeaderProps) {
  return (
    <header className="page-header">
      <div className="page-header-main">
        <div className="page-header-title">
          <h1>{title}</h1>
          {extra ? <span className="page-header-extra">{extra}</span> : null}
        </div>
        {description ? <p className="page-header-desc">{description}</p> : null}
      </div>
      {actions ? <div className="page-header-actions">{actions}</div> : null}
    </header>
  );
}
