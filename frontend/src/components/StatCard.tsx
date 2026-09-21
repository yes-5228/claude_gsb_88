// 看板指标卡。
import type { ReactNode } from 'react';

interface StatCardProps {
  label: string;
  value: ReactNode;
  hint?: string;
  tone?: 'default' | 'primary' | 'success' | 'warn' | 'danger';
  onClick?: () => void;
}

export function StatCard({ label, value, hint, tone = 'default', onClick }: StatCardProps) {
  return (
    <div
      className={`stat-card stat-${tone}${onClick ? ' stat-clickable' : ''}`}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
    >
      <p className="stat-label">{label}</p>
      <p className="stat-value">{value}</p>
      {hint ? <p className="stat-hint">{hint}</p> : null}
    </div>
  );
}
