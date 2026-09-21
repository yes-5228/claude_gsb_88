// 验收记录登记表单。
import { useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { acceptanceApi } from '../../api/acceptances';
import { recordApi } from '../../api/records';
import { taskApi } from '../../api/tasks';
import { FormField } from '../../components/FormField';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useForm, type FormErrors } from '../../hooks/useForm';
import { useMeta } from '../../providers/MetaProvider';
import type { AcceptancePayload } from '../../types/domain';
import { isDateString, today } from '../../utils/format';
import { optionLabel } from '../../utils/options';

/** 合格验收的最低评分，与后端 passScoreThreshold 保持一致。 */
const PASS_SCORE_THRESHOLD = 60;

interface AcceptanceFormValues {
  taskId: string;
  cleaningRecordId: string;
  acceptedAt: string;
  inspectorName: string;
  inspectorOrg: string;
  result: string;
  score: string;
  residualSludgeMm: string;
  issues: string;
  rectification: string;
  rectifyDeadline: string;
  remark: string;
}

function emptyForm(taskId = ''): AcceptanceFormValues {
  return {
    taskId,
    cleaningRecordId: '',
    acceptedAt: today(),
    inspectorName: '',
    inspectorOrg: '',
    result: 'pass',
    score: '90',
    residualSludgeMm: '0',
    issues: '',
    rectification: '',
    rectifyDeadline: '',
    remark: ''
  };
}

function toPayload(values: AcceptanceFormValues): AcceptancePayload {
  return {
    taskId: Number(values.taskId),
    cleaningRecordId: values.cleaningRecordId ? Number(values.cleaningRecordId) : null,
    acceptedAt: values.acceptedAt,
    inspectorName: values.inspectorName.trim(),
    inspectorOrg: values.inspectorOrg.trim(),
    result: values.result as AcceptancePayload['result'],
    score: Number(values.score),
    residualSludgeMm: values.residualSludgeMm === '' ? 0 : Number(values.residualSludgeMm),
    issues: values.issues.trim(),
    rectification: values.rectification.trim(),
    rectifyDeadline: values.result === 'rework' && values.rectifyDeadline ? values.rectifyDeadline : null,
    remark: values.remark.trim()
  };
}

function validate(values: AcceptanceFormValues): FormErrors<AcceptanceFormValues> {
  const errors: FormErrors<AcceptanceFormValues> = {};
  if (!values.taskId) {
    errors.taskId = '请选择待验收的清淤任务';
  }
  if (!values.acceptedAt) {
    errors.acceptedAt = '验收日期不能为空';
  } else if (!isDateString(values.acceptedAt)) {
    errors.acceptedAt = '验收日期格式应为 YYYY-MM-DD';
  } else if (values.acceptedAt > today()) {
    errors.acceptedAt = '验收日期不能晚于今天';
  }
  if (!values.inspectorName.trim()) {
    errors.inspectorName = '验收人不能为空';
  }
  if (!values.result) {
    errors.result = '请选择验收结论';
  }

  const score = Number(values.score);
  if (values.score === '' || !Number.isInteger(score) || score < 0 || score > 100) {
    errors.score = '验收评分需为 0 ~ 100 之间的整数';
  } else if (values.result === 'pass' && score < PASS_SCORE_THRESHOLD) {
    errors.score = `评分低于 ${PASS_SCORE_THRESHOLD} 分不能判定为合格，请选择需整改或修正评分`;
  }
  const residual = Number(values.residualSludgeMm);
  if (values.residualSludgeMm === '' || Number.isNaN(residual) || residual < 0 || residual > 1000) {
    errors.residualSludgeMm = '残留淤积厚度需在 0 ~ 1000 之间（mm）';
  }

  if (values.result === 'rework') {
    if (!values.issues.trim()) {
      errors.issues = '验收结论为需整改时，必须填写存在问题';
    }
    if (!values.rectifyDeadline) {
      errors.rectifyDeadline = '验收结论为需整改时，必须填写整改期限';
    } else if (values.acceptedAt && values.rectifyDeadline < values.acceptedAt) {
      errors.rectifyDeadline = '整改期限不能早于验收日期';
    }
  }
  return errors;
}

export function AcceptanceFormPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();

  const form = useForm<AcceptanceFormValues>(emptyForm(searchParams.get('taskId') ?? ''));
  const tasks = useAsync(() => taskApi.list({ pageSize: 100 }), []);

  const selectedTaskId = Number(form.values.taskId || '0');
  const records = useAsync(
    () => (selectedTaskId > 0 ? recordApi.list({ taskId: selectedTaskId, pageSize: 100 }) : Promise.resolve(null)),
    [selectedTaskId]
  );

  const [submitting, setSubmitting] = useState(false);

  const submit = () => {
    void form.handleSubmit(async () => {
      setSubmitting(true);
      try {
        const created = await acceptanceApi.create(toPayload(form.values));
        toast.success('验收记录已登记');
        navigate(`/acceptances/${created.id}`);
      } finally {
        setSubmitting(false);
      }
    }, validate);
  };

  // 只有「待验收」的任务可以登记验收。
  const assignable = (tasks.data?.list ?? []).filter((item) => item.status === 'completed');
  const currentTask = (tasks.data?.list ?? []).find((item) => item.id === selectedTaskId);
  const taskOptions =
    currentTask && !assignable.some((item) => item.id === currentTask.id) ? [currentTask, ...assignable] : assignable;
  const selectedTask = currentTask;
  const recordItems = records.data?.list ?? [];
  const isRework = form.values.result === 'rework';

  return (
    <form
      className="page"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <PageHeader
        title="登记验收记录"
        description="验收对象必须是已完成清淤并报验的任务；验收合格会同步把管段置为正常并累计清淤次数。"
        actions={
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
            返回
          </button>
        }
      />

      {form.serverError ? (
        <div className="alert alert-error">
          <p>{form.serverError}</p>
        </div>
      ) : null}

      {tasks.error ? (
        <div className="alert alert-warn">
          <p>任务下拉加载失败：{tasks.error}</p>
        </div>
      ) : null}

      <SectionCard
        title="验收对象"
        subtitle={`仅列出「待验收」的任务，共 ${assignable.length} 条；任务需先录入清淤记录再完工报验`}
        extra={<Link className="link" to="/tasks?status=completed">查看全部待验收任务</Link>}
      >
        <div className="form-grid">
          <FormField label="待验收任务" required span={2} error={form.errors.taskId}>
            <select
              className="select"
              value={form.values.taskId}
              onChange={(event) => form.setValue('taskId', event.target.value)}
            >
              <option value="">请选择任务</option>
              {taskOptions.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.code} · {item.title}（{optionLabel(enums?.taskStatuses, item.status)}）
                </option>
              ))}
            </select>
          </FormField>
          <FormField
            label="关联清淤记录"
            hint={
              selectedTaskId === 0
                ? '请先选择任务'
                : `该任务共 ${recordItems.length} 条清淤记录，可不指定`
            }
            error={form.errors.cleaningRecordId}
          >
            <select
              className="select"
              value={form.values.cleaningRecordId}
              onChange={(event) => form.setValue('cleaningRecordId', event.target.value)}
            >
              <option value="">不指定</option>
              {recordItems.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.code}（{item.cleanedAt ?? '未填日期'}）
                </option>
              ))}
            </select>
          </FormField>
        </div>
        {selectedTask ? (
          <>
            <div style={{ height: 12 }} />
            <div className="alert alert-info">
              <p>
                任务 {selectedTask.code} · 管段 {selectedTask.segment?.code ?? '—'}{' '}
                {selectedTask.segment?.name ?? ''} · 班组 {selectedTask.teamName || '—'} · 清淤记录{' '}
                {selectedTask.recordTotals?.recordCount ?? 0} 条 · 清淤量{' '}
                {selectedTask.recordTotals?.sludgeVolumeM3 ?? 0} m³
              </p>
            </div>
          </>
        ) : null}
      </SectionCard>

      <SectionCard title="验收结论" subtitle="带 * 的字段为必填项">
        <div className="form-grid">
          <FormField label="验收日期" required error={form.errors.acceptedAt}>
            <input
              className="input"
              type="date"
              value={form.values.acceptedAt}
              onChange={(event) => form.setValue('acceptedAt', event.target.value)}
            />
          </FormField>
          <FormField label="验收人" required error={form.errors.inspectorName}>
            <input
              className="input"
              value={form.values.inspectorName}
              onChange={(event) => form.setValue('inspectorName', event.target.value)}
            />
          </FormField>
          <FormField label="验收单位" error={form.errors.inspectorOrg}>
            <input
              className="input"
              value={form.values.inspectorOrg}
              onChange={(event) => form.setValue('inspectorOrg', event.target.value)}
            />
          </FormField>
          <FormField label="验收结论" required error={form.errors.result}>
            <select
              className="select"
              value={form.values.result}
              onChange={(event) => form.setValue('result', event.target.value)}
            >
              {(enums?.acceptanceResults ?? []).map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
          </FormField>
          <FormField
            label="验收评分"
            required
            hint={`合格判定不低于 ${PASS_SCORE_THRESHOLD} 分`}
            error={form.errors.score}
          >
            <input
              className="input"
              inputMode="numeric"
              value={form.values.score}
              onChange={(event) => form.setValue('score', event.target.value)}
            />
          </FormField>
          <FormField label="残留淤积厚度（mm）" error={form.errors.residualSludgeMm}>
            <input
              className="input"
              inputMode="decimal"
              value={form.values.residualSludgeMm}
              onChange={(event) => form.setValue('residualSludgeMm', event.target.value)}
            />
          </FormField>
          <FormField
            label="整改期限"
            hint={isRework ? '需整改时必填，且不早于验收日期' : '仅「需整改」结论需要填写'}
            error={form.errors.rectifyDeadline}
          >
            <input
              className="input"
              type="date"
              disabled={!isRework}
              value={form.values.rectifyDeadline}
              onChange={(event) => form.setValue('rectifyDeadline', event.target.value)}
            />
          </FormField>
          <FormField label="存在问题" span={3} error={form.errors.issues}>
            <textarea
              className="textarea"
              value={form.values.issues}
              placeholder="需整改时必填，例如：K0+320 处残留淤积厚度超标"
              onChange={(event) => form.setValue('issues', event.target.value)}
            />
          </FormField>
          <FormField label="整改要求" span={3} error={form.errors.rectification}>
            <textarea
              className="textarea"
              value={form.values.rectification}
              onChange={(event) => form.setValue('rectification', event.target.value)}
            />
          </FormField>
          <FormField label="备注" span={3} error={form.errors.remark}>
            <textarea
              className="textarea"
              value={form.values.remark}
              onChange={(event) => form.setValue('remark', event.target.value)}
            />
          </FormField>
        </div>

        <div className="form-actions">
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
            取消
          </button>
          <button type="submit" className="btn btn-primary" disabled={form.submitting || submitting}>
            {form.submitting || submitting ? '提交中…' : '提交验收结论'}
          </button>
        </div>
      </SectionCard>
    </form>
  );
}
