// 清淤记录详情：现场数据明细 + 修改留痕与版本对比 + 所属任务信息。
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
import { RecordVersionsCard } from './RecordVersionsCard';

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
  const editWindow = detail.data?.editWindow;
  const currentVersion = detail.data?.currentVersion ?? 1;
  const editable = editWindow?.open ?? false;

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
        description="清淤记录是验收结论的计量依据，修改窗口内可调整并全程留痕，所有统计按最新版本计算。"
        extra={
          <>
            <span className="tag tag-info">当前版本 v{currentVersion}</span>
            {task ? <StatusTag list="taskStatuses" value={task.status} /> : null}
          </>
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
              onClick={() => navigate(`/records/${id}/edit`)}
            >
              编辑
            </button>
            <button
              type="button"
              className="btn btn-danger"
              disabled={!record || !editable}
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
                  修改窗口开放中：任务处于「待开工」或「清淤中」时可调整记录，每次修改都会保存修改前的数值、
                  修改人、时间与原因；任务完工报验后窗口关闭，被验收引用的记录只能查看历史版本。
                </p>
              </div>
            ) : (
              <div className="alert alert-warn">
                <p>{editWindow?.reason || '修改窗口已关闭，该记录只能查看历史版本。'}</p>
              </div>
            )}

            <SectionCard title="现场数据" subtitle={`录入于 ${formatDateTime(record.createdAt)}，最近更新 ${formatDateTime(record.updatedAt)}`}>
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

            <RecordVersionsCard recordId={id} />

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
        message={<p>删除后该条清淤数据及其全部修改留痕将从任务汇总中扣除，且不可恢复。</p>}
        onConfirm={handleDelete}
        onCancel={() => setConfirmOpen(false)}
      />
    </div>
  );
}
