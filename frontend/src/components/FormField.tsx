// 表单字段容器：统一标签、必填标记、提示与错误展示。
import type { ReactNode } from 'react';

interface FormFieldProps {
  label: string;
  required?: boolean;
  hint?: string;
  error?: string;
  span?: 1 | 2 | 3;
  children: ReactNode;
}

export function FormField({ label, required, hint, error, span = 1, children }: FormFieldProps) {
  return (
    <label className={`form-field form-span-${span}`}>
      <span className="form-label">
        {label}
        {required ? <em className="form-required">*</em> : null}
      </span>
      {children}
      {hint ? <span className="form-hint">{hint}</span> : null}
      {error ? <span className="form-error">{error}</span> : null}
    </label>
  );
}
