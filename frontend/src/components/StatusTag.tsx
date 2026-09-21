// 状态标签：统一按枚举字典翻译中文，并按语义着色。
import { useMeta } from '../providers/MetaProvider';
import type { Enums } from '../types/domain';
import { optionLabel } from '../utils/options';

/** 需要翻译的枚举分组，直接对应 /meta/enums 的字段名。 */
export type TagList =
  | 'taskStatuses'
  | 'segmentStatuses'
  | 'taskPriorities'
  | 'taskSources'
  | 'cleaningMethods'
  | 'weathers'
  | 'acceptanceResults'
  | 'pipeTypes';

const TONES: Record<string, string> = {
  pending: 'muted',
  in_progress: 'info',
  completed: 'warn',
  accepted: 'success',
  cancelled: 'muted',
  normal: 'success',
  attention: 'warn',
  blocked: 'danger',
  low: 'muted',
  high: 'warn',
  urgent: 'danger',
  pass: 'success',
  rework: 'danger'
};

interface StatusTagProps {
  list: TagList;
  value: string | null | undefined;
}

export function StatusTag({ list, value }: StatusTagProps) {
  const { enums } = useMeta();
  if (!value) {
    return <span className="tag tag-muted">—</span>;
  }
  const tone = TONES[value] ?? 'info';
  return <span className={`tag tag-${tone}`}>{optionLabel(enums?.[list], value)}</span>;
}

/** 供页面按任意枚举分组取中文名。 */
export function useEnumLabels(list: keyof Enums): (value: string | null | undefined) => string {
  const { enums } = useMeta();
  return (value) => optionLabel(enums?.[list], value);
}
