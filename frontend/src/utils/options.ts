// 枚举字典与业务值之间的转换工具。
import type { Option } from '../types/domain';

/** 把枚举值翻译成中文标签，找不到时回退为原值。 */
export function optionLabel(options: Option[] | undefined | null, value: string | null | undefined): string {
  if (!value) {
    return '—';
  }
  const matched = options?.find((item) => item.value === value);
  return matched ? matched.label : value;
}

/** 选项列表兜底，避免渲染时出现 undefined。 */
export function safeOptions(options: Option[] | undefined | null): Option[] {
  return options ?? [];
}
