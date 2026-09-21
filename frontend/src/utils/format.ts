// 日期与数字的统一展示口径，避免每个页面各写一套格式化逻辑。
import dayjs from 'dayjs';

/** 展示日期（YYYY-MM-DD），空值统一显示占位符。 */
export function formatDate(value: string | null | undefined): string {
  if (!value) {
    return '—';
  }
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format('YYYY-MM-DD') : '—';
}

/** 展示日期时间（精确到分钟），用于 createdAt / updatedAt 这类审计字段。 */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return '—';
  }
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : '—';
}

/** 表单默认值：今天的 YYYY-MM-DD。 */
export function today(): string {
  return dayjs().format('YYYY-MM-DD');
}

/** 数字展示：保留指定小数位并去掉无意义的尾随 0。 */
export function formatNumber(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '—';
  }
  if (digits <= 0) {
    return String(Math.round(value));
  }
  const fixed = value.toFixed(digits);
  if (!fixed.includes('.')) {
    return fixed;
  }
  return fixed.replace(/0+$/, '').replace(/\.$/, '');
}

/** 长度展示（米）。 */
export function formatLength(value: number | null | undefined): string {
  if (value === null || value === undefined) {
    return '—';
  }
  return `${formatNumber(value, 2)} m`;
}

/** 体积展示（立方米）。 */
export function formatVolume(value: number | null | undefined): string {
  if (value === null || value === undefined) {
    return '—';
  }
  return `${formatNumber(value, 2)} m³`;
}

/**
 * 百分比展示。
 *
 * 后端 acceptancePassRate 等字段已经按百分数返回（66.67 表示 66.67%），
 * 因此这里不再做乘 100 的换算，避免出现 6667% 这类错误展示。
 */
export function formatPercent(value: number | null | undefined): string {
  if (value === null || value === undefined) {
    return '—';
  }
  return `${formatNumber(value, 1)}%`;
}

/** 判断字符串是否为合法的 YYYY-MM-DD。 */
export function isDateString(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value) && dayjs(value, 'YYYY-MM-DD').isValid();
}
