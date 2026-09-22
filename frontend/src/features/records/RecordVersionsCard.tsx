// 清淤记录修改留痕：版本列表 + 任意两个版本的字段级对比。
import { useEffect, useMemo, useState } from 'react';
import { recordApi } from '../../api/records';
import { DataTable, type Column } from '../../components/DataTable';
import { SectionCard } from '../../components/SectionCard';
import { useAsync } from '../../hooks/useAsync';
import { useMeta } from '../../providers/MetaProvider';
import type { RecordVersion } from '../../types/domain';
import { formatDate, formatDateTime, formatLength, formatNumber, formatVolume } from '../../utils/format';
import { optionLabel } from '../../utils/options';

interface RecordVersionsCardProps {
  recordId: number;
}

interface VersionField {
  key: keyof RecordVersion;
  label: string;
  format: (version: RecordVersion) => string;
}

/** 版本标签：当前版本额外标注。 */
function versionLabel(version: RecordVersion): string {
  return version.current ? `v${version.version} · 当前` : `v${version.version}`;
}

export function RecordVersionsCard({ recordId }: RecordVersionsCardProps) {
  const { enums } = useMeta();
  const versionsQuery = useAsync(() => recordApi.versions(recordId), [recordId]);

  const versions = useMemo(
    () => [...(versionsQuery.data?.versions ?? [])].sort((a, b) => b.version - a.version),
    [versionsQuery.data]
  );

  // 对比的两个版本：默认上一版本 vs 当前版本。
  const [baseVersion, setBaseVersion] = useState(0);
  const [compareVersion, setCompareVersion] = useState(0);
  const currentVersion = versionsQuery.data?.currentVersion ?? 0;

  useEffect(() => {
    if (currentVersion > 0 && compareVersion === 0) {
      setCompareVersion(currentVersion);
      setBaseVersion(currentVersion > 1 ? currentVersion - 1 : currentVersion);
    }
  }, [currentVersion, compareVersion]);

  const fields = useMemo<VersionField[]>(
    () => [
      { key: 'cleanedAt', label: '清淤日期', format: (v) => formatDate(v.cleanedAt) },
      { key: 'lengthM', label: '清淤长度', format: (v) => formatLength(v.lengthM) },
      { key: 'sludgeVolumeM3', label: '清淤量', format: (v) => formatVolume(v.sludgeVolumeM3) },
      { key: 'waterVolumeM3', label: '用水量', format: (v) => formatVolume(v.waterVolumeM3) },
      { key: 'personnelCount', label: '作业人数', format: (v) => `${formatNumber(v.personnelCount, 0)} 人` },
      { key: 'method', label: '清淤方式', format: (v) => optionLabel(enums?.cleaningMethods, v.method) },
      { key: 'weather', label: '天气', format: (v) => optionLabel(enums?.weathers, v.weather) },
      { key: 'equipment', label: '主要设备', format: (v) => v.equipment || '—' },
      { key: 'sludgeDisposalSite', label: '污泥消纳点', format: (v) => v.sludgeDisposalSite || '—' },
      { key: 'recorderName', label: '记录人', format: (v) => v.recorderName || '—' },
      { key: 'safetyMeasures', label: '安全措施', format: (v) => v.safetyMeasures || '—' },
      { key: 'problemFound', label: '发现的问题', format: (v) => v.problemFound || '—' },
      { key: 'remark', label: '备注', format: (v) => v.remark || '—' }
    ],
    [enums]
  );

  const base = versions.find((item) => item.version === baseVersion) ?? null;
  const compare = versions.find((item) => item.version === compareVersion) ?? null;

  const columns: Column<RecordVersion>[] = [
    {
      key: 'version',
      title: '版本',
      width: '110px',
      render: (row) => (
        <>
          <span className="cell-main">v{row.version}</span>{' '}
          {row.current ? <span className="tag tag-info">当前版本</span> : null}
        </>
      )
    },
    {
      key: 'changedAt',
      title: '修改时间',
      width: '150px',
      render: (row) => (row.current ? '—' : formatDateTime(row.changedAt))
    },
    {
      key: 'editorName',
      title: '修改人',
      width: '100px',
      render: (row) => row.editorName || '—'
    },
    {
      key: 'changeReason',
      title: '修改原因',
      render: (row) => row.changeReason || '—'
    }
  ];

  return (
    <SectionCard
      title="修改留痕"
      subtitle="每次修改都会保存修改前的数值、修改人、时间与原因；选择任意两个版本即可对比差异"
    >
      <DataTable
        columns={columns}
        rows={versions}
        rowKey={(row) => row.version}
        loading={versionsQuery.loading}
        error={versionsQuery.error}
        onRetry={versionsQuery.reload}
        emptyText="暂无版本记录"
      />

      {versions.length > 1 ? (
        <div className="version-compare">
          <div className="compare-bar">
            <div className="filter-item">
              <span className="filter-label">基准版本</span>
              <select
                className="select"
                value={baseVersion}
                onChange={(event) => setBaseVersion(Number(event.target.value))}
              >
                {versions.map((item) => (
                  <option key={item.version} value={item.version}>
                    {versionLabel(item)}
                  </option>
                ))}
              </select>
            </div>
            <div className="filter-item">
              <span className="filter-label">对比版本</span>
              <select
                className="select"
                value={compareVersion}
                onChange={(event) => setCompareVersion(Number(event.target.value))}
              >
                {versions.map((item) => (
                  <option key={item.version} value={item.version}>
                    {versionLabel(item)}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {base && compare && base.version !== compare.version ? (
            <div className="table-wrap">
              <table className="data-table diff-table">
                <thead>
                  <tr>
                    <th style={{ width: '160px' }}>字段</th>
                    <th>{versionLabel(base)}</th>
                    <th>{versionLabel(compare)}</th>
                  </tr>
                </thead>
                <tbody>
                  {fields.map((field) => {
                    const changed = base[field.key] !== compare[field.key];
                    return (
                      <tr key={field.key} className={changed ? 'diff-changed' : undefined}>
                        <td>{field.label}</td>
                        <td className={changed ? 'diff-value-old' : undefined}>{field.format(base)}</td>
                        <td className={changed ? 'diff-value-new' : undefined}>{field.format(compare)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="form-note">请选择两个不同的版本进行对比。</p>
          )}
        </div>
      ) : null}
    </SectionCard>
  );
}
