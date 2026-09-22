// 清淤记录的版本履历与任意两版差异对比。
//
// 履历数据来自 GET /cleaning-records/:id/revisions：
// - 顶部按时间线展示每个版本（动作、修改人、时间、原因）；
// - 下方可任选两个版本，逐字段对比差异，默认对比「上一版 → 当前版」。
import { useMemo, useState } from 'react';
import { recordApi } from '../../api/records';
import { Modal } from '../../components/Modal';
import { SectionCard } from '../../components/SectionCard';
import { StateBlock } from '../../components/StateBlock';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { RecordRevision, RecordVersionSnapshot } from '../../types/domain';
import { formatDate, formatDateTime, formatLength, formatNumber, formatVolume } from '../../utils/format';
import { optionLabel } from '../../utils/options';

type FieldKey = keyof RecordVersionSnapshot;

interface FieldMeta {
  key: FieldKey;
  label: string;
  type: 'date' | 'length' | 'volume' | 'people' | 'method' | 'weather' | 'text';
}

const FIELD_METAS: FieldMeta[] = [
  { key: 'cleanedAt', label: '清淤日期', type: 'date' },
  { key: 'method', label: '清淤方式', type: 'method' },
  { key: 'lengthM', label: '清淤长度', type: 'length' },
  { key: 'sludgeVolumeM3', label: '清淤量', type: 'volume' },
  { key: 'waterVolumeM3', label: '用水量', type: 'volume' },
  { key: 'personnelCount', label: '作业人数', type: 'people' },
  { key: 'weather', label: '天气', type: 'weather' },
  { key: 'equipment', label: '主要设备', type: 'text' },
  { key: 'sludgeDisposalSite', label: '污泥消纳点', type: 'text' },
  { key: 'recorderName', label: '记录人', type: 'text' },
  { key: 'safetyMeasures', label: '安全措施', type: 'text' },
  { key: 'problemFound', label: '发现的问题', type: 'text' },
  { key: 'remark', label: '备注', type: 'text' }
];

function isChanged(key: FieldKey, a: RecordVersionSnapshot, b: RecordVersionSnapshot): boolean {
  const valueA = a[key];
  const valueB = b[key];
  if (typeof valueA === 'number' || typeof valueB === 'number') {
    return Number(valueA) !== Number(valueB);
  }
  return String(valueA ?? '').trim() !== String(valueB ?? '').trim();
}

const TEXT_EMPTY = '—';

function formatValue(meta: FieldMeta, snapshot: RecordVersionSnapshot | undefined, enums: ReturnType<typeof useMeta>['enums']): string {
  if (!snapshot) {
    return TEXT_EMPTY;
  }
  const value = snapshot[meta.key];
  switch (meta.type) {
    case 'date':
      return formatDate(value as string | null);
    case 'length':
      return formatLength(value as number);
    case 'volume':
      return formatVolume(value as number);
    case 'people':
      return `${formatNumber(value as number, 0)} 人`;
    case 'method':
      return optionLabel(enums?.cleaningMethods, value as string) || TEXT_EMPTY;
    case 'weather':
      return optionLabel(enums?.weathers, value as string) || TEXT_EMPTY;
    default: {
      const text = String(value ?? '').trim();
      return text === '' ? TEXT_EMPTY : text;
    }
  }
}

function versionCaption(item: RecordRevision | undefined): string {
  if (!item) {
    return '未选择';
  }
  const action = item.action === 'create' ? '录入' : '修改';
  return `v${item.version}（${action}）`;
}

interface RevisionPanelProps {
  recordId: number;
  /** 最新版本号，用于在履历加载前提示。 */
  latestVersion: number;
}

export function RecordRevisionPanel({ recordId, latestVersion }: RevisionPanelProps) {
  const { enums } = useMeta();
  const history = useAsync(() => recordApi.revisions(recordId), [recordId]);
  const items = history.data?.items ?? [];

  // 对比弹窗状态。
  const [compareOpen, setCompareOpen] = useState(false);
  const [baseVersion, setBaseVersion] = useState<number | null>(null);
  const [targetVersion, setTargetVersion] = useState<number | null>(null);

  const openCompare = () => {
    // 默认对比「上一版 → 当前版」；只有一个版本时两栏都选它。
    const latest = items[0]?.version ?? latestVersion;
    const previous = items[1]?.version ?? latest;
    setBaseVersion(previous);
    setTargetVersion(latest);
    setCompareOpen(true);
  };

  const base = useMemo(() => items.find((item) => item.version === baseVersion), [items, baseVersion]);
  const target = useMemo(() => items.find((item) => item.version === targetVersion), [items, targetVersion]);

  const changedCount = useMemo(() => {
    if (!base || !target) {
      return 0;
    }
    return FIELD_METAS.reduce((count, meta) => count + (isChanged(meta.key, base.snapshot, target.snapshot) ? 1 : 0), 0);
  }, [base, target]);

  return (
    <SectionCard
      title="修改履历与版本对比"
      subtitle="每次调整都会保存修改前数值、修改人、时间与原因，可任选两个版本对比差异。"
      extra={
        <button type="button" className="btn btn-ghost btn-sm" disabled={items.length === 0} onClick={openCompare}>
          对比版本差异
        </button>
      }
    >
      <StateBlock loading={history.loading} error={history.error} onRetry={history.reload} empty={items.length === 0} emptyText="暂无版本履历">
        <ol className="revision-timeline">
          {items.map((item) => (
            <li key={item.version} className="revision-item">
              <div className="revision-head">
                <span className={`revision-tag ${item.action === 'create' ? 'revision-tag-create' : 'revision-tag-update'}`}>
                  {item.action === 'create' ? '首次录入' : '修改留痕'}
                </span>
                <span className="revision-version">v{item.version}</span>
                <span className="revision-meta">
                  {item.changedBy} · {formatDateTime(item.changedAt)}
                </span>
              </div>
              {item.action === 'update' ? <p className="revision-reason">原因：{item.changeReason || '—'}</p> : null}
            </li>
          ))}
        </ol>
      </StateBlock>

      <Modal
        open={compareOpen}
        title="对比清淤记录版本差异"
        width={860}
        onClose={() => setCompareOpen(false)}
        footer={
          <button type="button" className="btn btn-primary" onClick={() => setCompareOpen(false)}>
            关闭
          </button>
        }
      >
        {items.length === 0 ? (
          <p className="form-note">暂无可对比的版本。</p>
        ) : (
          <div className="revision-compare">
            <div className="compare-pickers">
              <label className="compare-picker">
                <span>基准版本</span>
                <select className="select" value={baseVersion ?? ''} onChange={(event) => setBaseVersion(Number(event.target.value))}>
                  {[...items].reverse().map((item) => (
                    <option key={item.version} value={item.version}>
                      v{item.version} · {item.action === 'create' ? '录入' : '修改'} · {formatDateTime(item.changedAt)}
                    </option>
                  ))}
                </select>
              </label>
              <span className="compare-arrow" aria-hidden="true">
                →
              </span>
              <label className="compare-picker">
                <span>对比版本</span>
                <select className="select" value={targetVersion ?? ''} onChange={(event) => setTargetVersion(Number(event.target.value))}>
                  {[...items].reverse().map((item) => (
                    <option key={item.version} value={item.version}>
                      v{item.version} · {item.action === 'create' ? '录入' : '修改'} · {formatDateTime(item.changedAt)}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            <p className={`compare-summary ${changedCount > 0 ? 'compare-summary-changed' : ''}`}>
              {baseVersion === targetVersion
                ? '当前选择的是同一个版本，请选择两个不同的版本查看差异。'
                : changedCount > 0
                  ? `共 ${changedCount} 个字段发生变化（${versionCaption(base)} → ${versionCaption(target)}）`
                  : `两个版本内容一致，没有字段差异（${versionCaption(base)} → ${versionCaption(target)}）`}
            </p>

            <div className="table-wrap">
              <table className="data-table diff-table">
                <thead>
                  <tr>
                    <th style={{ width: 130 }}>字段</th>
                    <th>{versionCaption(base)}</th>
                    <th>{versionCaption(target)}</th>
                  </tr>
                </thead>
                <tbody>
                  {FIELD_METAS.map((meta) => {
                    const changed = !!base && !!target && baseVersion !== targetVersion && isChanged(meta.key, base.snapshot, target.snapshot);
                    return (
                      <tr key={meta.key} className={changed ? 'diff-row-changed' : undefined}>
                        <th className="diff-field">
                          {meta.label}
                          {changed ? <span className="diff-badge">已变更</span> : null}
                        </th>
                        <td className={changed ? 'diff-old' : undefined}>{formatValue(meta, base?.snapshot, enums)}</td>
                        <td className={changed ? 'diff-new' : undefined}>{formatValue(meta, target?.snapshot, enums)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </Modal>
    </SectionCard>
  );
}

