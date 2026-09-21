// 清淤任务登记 / 编辑表单。
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { segmentApi } from '../../api/pipesegments';
import { taskApi } from '../../api/tasks';
import { FormField } from '../../components/FormField';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { StateBlock } from '../../components/StateBlock';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useForm, type FormErrors } from '../../hooks/useForm';
import { useMeta } from '../../providers/MetaProvider';
import type { CleaningTask, TaskPayload } from '../../types/domain';

interface TaskFormValues {
  title: string;
  pipeSegmentId: string;
  priority: string;
  source: string;
  method: string;
  planStartDate: string;
  planEndDate: string;
  teamName: string;
  leaderName: string;
  leaderPhone: string;
  description: string;
}

const EMPTY_FORM: TaskFormValues = {
  title: '',
  pipeSegmentId: '',
  priority: 'normal',
  source: 'plan',
  method: '',
  planStartDate: '',
  planEndDate: '',
  teamName: '',
  leaderName: '',
  leaderPhone: '',
  description: ''
};

function toFormValues(task: CleaningTask): TaskFormValues {
  return {
    title: task.title,
    pipeSegmentId: String(task.pipeSegmentId),
    priority: task.priority,
    source: task.source,
    method: task.method,
    planStartDate: task.planStartDate ?? '',
    planEndDate: task.planEndDate ?? '',
    teamName: task.teamName,
    leaderName: task.leaderName,
    leaderPhone: task.leaderPhone,
    description: task.description
  };
}

function toPayload(values: TaskFormValues): TaskPayload {
  return {
    title: values.title.trim(),
    pipeSegmentId: Number(values.pipeSegmentId),
    priority: values.priority as TaskPayload['priority'],
    source: values.source as TaskPayload['source'],
    method: values.method as TaskPayload['method'],
    planStartDate: values.planStartDate,
    planEndDate: values.planEndDate,
    teamName: values.teamName.trim(),
    leaderName: values.leaderName.trim(),
    leaderPhone: values.leaderPhone.trim(),
    description: values.description.trim()
  };
}

/** 联系电话允许数字、空格、加号与横线，长度 6-32 位。 */
const PHONE_PATTERN = /^[0-9+ -]{6,32}$/;

function validate(values: TaskFormValues): FormErrors<TaskFormValues> {
  const errors: FormErrors<TaskFormValues> = {};
  if (!values.title.trim()) {
    errors.title = '任务标题不能为空';
  }
  if (!values.pipeSegmentId) {
    errors.pipeSegmentId = '请选择关联管段';
  }
  if (!values.planStartDate) {
    errors.planStartDate = '计划开始日期不能为空';
  }
  if (!values.planEndDate) {
    errors.planEndDate = '计划完成日期不能为空';
  } else if (values.planStartDate && values.planEndDate < values.planStartDate) {
    errors.planEndDate = '计划完成日期不能早于计划开始日期';
  }
  if (values.leaderPhone.trim() && !PHONE_PATTERN.test(values.leaderPhone.trim())) {
    errors.leaderPhone = '联系电话只能包含数字、空格、加号和横线，长度 6-32 位';
  }
  return errors;
}

export function TaskFormPage() {
  const params = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();
  const id = Number(params.id ?? '0');
  const isEdit = id > 0;

  const form = useForm<TaskFormValues>(EMPTY_FORM);
  const [hydrated, setHydrated] = useState(false);

  const detail = useAsync(() => (isEdit ? taskApi.detail(id) : Promise.resolve(null)), [id, isEdit]);
  const segments = useAsync(() => segmentApi.options(), []);

  useEffect(() => {
    const task = detail.data?.task;
    if (task && !hydrated) {
      form.reset(toFormValues(task));
      setHydrated(true);
    }
  }, [detail.data, hydrated, form]);

  const submit = () => {
    void form.handleSubmit(async () => {
      const payload = toPayload(form.values);
      if (isEdit) {
        await taskApi.update(id, payload);
        toast.success('清淤任务已保存');
        navigate(`/tasks/${id}`);
      } else {
        const created = await taskApi.create(payload);
        toast.success('清淤任务已登记');
        navigate(`/tasks/${created.id}`);
      }
    }, validate);
  };

  const segmentItems = segments.data?.items ?? [];

  return (
    <form
      className="page"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <PageHeader
        title={isEdit ? '编辑清淤任务' : '登记清淤任务'}
        description="任务必须关联一个管段；只有待开工或清淤中的任务可以修改，已完工报验后需通过验收流程回退。"
        actions={
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
            返回
          </button>
        }
      />

      <StateBlock loading={isEdit && detail.loading} error={isEdit ? detail.error : ''} onRetry={detail.reload}>
        {form.serverError ? (
          <div className="alert alert-error">
            <p>{form.serverError}</p>
          </div>
        ) : null}

        {segments.error ? (
          <div className="alert alert-warn">
            <p>管段下拉加载失败：{segments.error}</p>
          </div>
        ) : null}

        <SectionCard title="任务信息" subtitle="带 * 的字段为必填项">
          <div className="form-grid">
            <FormField label="任务标题" required span={2} error={form.errors.title}>
              <input
                className="input"
                value={form.values.title}
                placeholder="例如 解放路雨水管段汛前清淤"
                onChange={(event) => form.setValue('title', event.target.value)}
              />
            </FormField>
            <FormField
              label="关联管段"
              required
              error={form.errors.pipeSegmentId}
              hint={`共 ${segmentItems.length} 个可选管段`}
            >
              <select
                className="select"
                value={form.values.pipeSegmentId}
                onChange={(event) => form.setValue('pipeSegmentId', event.target.value)}
              >
                <option value="">请选择管段</option>
                {segmentItems.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.code} · {item.name}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="优先级" error={form.errors.priority}>
              <select
                className="select"
                value={form.values.priority}
                onChange={(event) => form.setValue('priority', event.target.value)}
              >
                {(enums?.taskPriorities ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="任务来源" error={form.errors.source}>
              <select
                className="select"
                value={form.values.source}
                onChange={(event) => form.setValue('source', event.target.value)}
              >
                {(enums?.taskSources ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="计划清淤方式" hint="可留空，实际方式以清淤记录为准" error={form.errors.method}>
              <select
                className="select"
                value={form.values.method}
                onChange={(event) => form.setValue('method', event.target.value)}
              >
                <option value="">未确定</option>
                {(enums?.cleaningMethods ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="计划开始日期" required error={form.errors.planStartDate}>
              <input
                className="input"
                type="date"
                value={form.values.planStartDate}
                onChange={(event) => form.setValue('planStartDate', event.target.value)}
              />
            </FormField>
            <FormField label="计划完成日期" required error={form.errors.planEndDate}>
              <input
                className="input"
                type="date"
                value={form.values.planEndDate}
                onChange={(event) => form.setValue('planEndDate', event.target.value)}
              />
            </FormField>
            <FormField label="实施班组" error={form.errors.teamName}>
              <input
                className="input"
                value={form.values.teamName}
                onChange={(event) => form.setValue('teamName', event.target.value)}
              />
            </FormField>
            <FormField label="现场负责人" error={form.errors.leaderName}>
              <input
                className="input"
                value={form.values.leaderName}
                onChange={(event) => form.setValue('leaderName', event.target.value)}
              />
            </FormField>
            <FormField label="联系电话" error={form.errors.leaderPhone}>
              <input
                className="input"
                value={form.values.leaderPhone}
                onChange={(event) => form.setValue('leaderPhone', event.target.value)}
              />
            </FormField>
            <FormField label="任务说明" span={3} error={form.errors.description}>
              <textarea
                className="textarea"
                value={form.values.description}
                placeholder="补充任务背景、作业范围或安全要求"
                onChange={(event) => form.setValue('description', event.target.value)}
              />
            </FormField>
          </div>

          <div className="form-actions">
            <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
              取消
            </button>
            <button type="submit" className="btn btn-primary" disabled={form.submitting}>
              {form.submitting ? '保存中…' : '保存'}
            </button>
          </div>
        </SectionCard>
      </StateBlock>
    </form>
  );
}
