// 内容卡片，页面按业务区块拆分展示。
import type { ReactNode } from 'react';

interface SectionCardProps {
  title: string;
  subtitle?: string;
  extra?: ReactNode;
  children: ReactNode;
}

export function SectionCard({ title, subtitle, extra, children }: SectionCardProps) {
  return (
    <section className="card">
      <header className="card-header">
        <div>
          <h2>{title}</h2>
          {subtitle ? <p className="card-subtitle">{subtitle}</p> : null}
        </div>
        {extra ? <div className="card-extra">{extra}</div> : null}
      </header>
      <div className="card-body">{children}</div>
    </section>
  );
}
