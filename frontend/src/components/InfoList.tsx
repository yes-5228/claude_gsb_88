// 详情页的键值信息网格。
import type { ReactNode } from 'react';

export interface InfoItem {
  label: string;
  value: ReactNode;
  span?: 1 | 2 | 3;
}

interface InfoListProps {
  items: InfoItem[];
}

export function InfoList({ items }: InfoListProps) {
  return (
    <dl className="info-list">
      {items.map((item) => (
        <div key={item.label} className={`info-item info-span-${item.span ?? 1}`}>
          <dt>{item.label}</dt>
          <dd>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}
