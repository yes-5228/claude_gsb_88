// 清淤记录详情：现场数据明细 + 所属任务信息 + 修改履历与版本对比。
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { toErrorMessage } from '../../api/client';
import { recordApi } from '../../api/records';
import { ConfirmDialog } from '../../components/ConfirmDialog';
import { InfoList } from '../../components/InfoList';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { StatusTag } from '../../components/StatusTag';
import { StateBlock } from '../../components/StateBlock';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { formatDate, formatDateTime, formatLength, formatNumber, formatVolume } from '../../utils/format';
import { RecordRevisionPanel } from './RecordRevisionPanel';

export function RecordDetailPage() {
  const params = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const id = Number(params.id ?? '0');

  const detail = useAsync(
    () => (id > 0 ? recordApi.detail(id) : Promise.reject(new Error('记录编号无效'))),
    [id]
  );

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const record = detail.data?.record;
  const task = detail.data?.task;
  const editable = detail.data?.editable ?? false;
  const lockedReason = detail.data?.lockedReason ?? '';

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await recordApi.remove(id);
      toast.success('清淤记录已删除');
      navigate('/records');
    } catch (cause: unknown) {
      toast.error(toErrorMessage(cause));
      setConfirmOpen(false);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="page">
      <PageHeader
        title={record ? `${record.code} 清淤记录` : '清淤记录详情'}
        description="清淤记录是验收结论的计量依据；每次修改都会生成新版本，履历可查、版本可对比。"
        extra={
          record ? (
            <div className="header-tags">
              <StatusTag list="taskStatuses" value={task?.status} />
              <span className="tag tag-info">v{record.version}</span>
            </div>
          ) : null
        }
        actions={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => navigate('/records')}>
              返回列表
            </button>
            <button
              type="button"
              className="btn btn-ghost"
              disabled={!record || !editable}
              title={!editable ? lockedReason : '调整记录并生成新版本'}
              onClick={() => navigate(`/records/${id}/edit`)}
            >
              修改为新版本
            </button>
            <button
              type="button"
              className="btn btn-danger"
              disabled={!record || !editable}
              title={!editable ? lockedReason : ''}
              onClick={() => setConfirmOpen(true)}
            >
              删除
            </button>
          </>
        }
      />

      <StateBlock loading={detail.loading} error={detail.error} onRetry={detail.reload} empty={!record} emptyText="清淤记录不存在">
        {record ? (
          <>
            {editable ? (
              <div className="alert alert-info">
                <p>
                  修改窗口开放中：任务处于「待开工」或「清淤中」环节时可以继续调整；
                  每次修改都会保存修改前数值、修改人、时间与原因，履历、汇总与看板数字同步刷新。
                </p>
              </div>
            ) : (
              <div className="alert alert-warn">
                <p>{lockedReason || '该记录当前不可修改。'}</p>
                <p>记录数值仍可在下方「修改履历与版本对比」中查看全部历史版本并逐版对比。</p>
              </div>
            )}

            <SectionCard
              title="现场数据（最新版本）"
              subtitle={`当前为 v${record.version}，录入于 ${formatDateTime(record.createdAt)}，最近更新 ${formatDateTime(record.updatedAt)}`}
            >
              <InfoList
                items={[
                  { label: '记录编号', value: record.code },
                  { label: '清淤日期', value: formatDate(record.cleanedAt) },
                  { label: '清淤方式', value: <StatusTag list="cleaningMethods" value={record.method} /> },
                  { label: '清淤长度', value: formatLength(record.lengthM) },
                  { label: '清淤量', value: formatVolume(record.sludgeVolumeM3) },
                  { label: '用水量', value: formatVolume(record.waterVolumeM3) },
                  { label: '作业人数', value: `${formatNumber(record.personnelCount, 0)} 人` },
                  { label: '天气', value: <StatusTag list="weathers" value={record.weather} /> },
                  { label: '主要设备', value: record.equipment || '—' },
                  { label: '污泥消纳点', value: record.sludgeDisposalSite || '—' },
                  { label: '记录人', value: record.recorderName || '—' },
                  { label: '安全措施', value: record.safetyMeasures || '—', span: 3 },
                  { label: '发现的问题', value: record.problemFound || '—', span: 3 },
                  { label: '备注', value: record.remark || '—', span: 3 }
                ]}
              />
            </SectionCard>

            <RecordRevisionPanel recordId={id} latestVersion={record.version} />

            <SectionCard
              title="所属任务"
              subtitle="清淤记录必须归属一个清淤任务"
              extra={
                task ? (
                  <Link className="link" to={`/tasks/${task.id}`}>
                    查看任务
                  </Link>
                ) : null
              }
            >
              {task ? (
                <InfoList
                  items={[
                    { label: '任务编号', value: task.code },
                    { label: '任务标题', value: task.title },
                    { label: '任务状态', value: <StatusTag list="taskStatuses" value={task.status} /> },
                    { label: '优先级', value: <StatusTag list="taskPriorities" value={task.priority} /> },
                    { label: '实施班组', value: task.teamName || '—' },
                    { label: '关联管段', value: `${task.segmentCode} · ${task.segmentName}` },
                    { label: '所属片区', value: task.segmentDistrict || '—' }
                  ]}
                />
              ) : (
                <p className="form-note">关联任务已不存在。</p>
              )}
            </SectionCard>
          </>
        ) : null}
      </StateBlock>

      <ConfirmDialog
        open={confirmOpen}
        title="删除清淤记录"
        danger
        busy={deleting}
        confirmText="确认删除"
        message={<p>删除后该条清淤数据及其版本履历将从任务汇总中扣除，且不可恢复。</p>}
        onConfirm={handleDelete}
        onCancel={() => setConfirmOpen(false)}
      />
    </div>
  );
}
