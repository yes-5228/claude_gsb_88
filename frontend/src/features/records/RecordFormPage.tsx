// 清淤记录录入 / 编辑表单。
import { useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { recordApi } from '../../api/records';
import { taskApi } from '../../api/tasks';
import { FormField } from '../../components/FormField';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { StateBlock } from '../../components/StateBlock';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useForm, type FormErrors } from '../../hooks/useForm';
import { useMeta } from '../../providers/MetaProvider';
import type { CleaningRecord, RecordPayload } from '../../types/domain';
import { isDateString, today } from '../../utils/format';
import { optionLabel } from '../../utils/options';

interface RecordFormValues {
  taskId: string;
  cleanedAt: string;
  lengthM: string;
  sludgeVolumeM3: string;
  waterVolumeM3: string;
  personnelCount: string;
  method: string;
  equipment: string;
  weather: string;
  sludgeDisposalSite: string;
  safetyMeasures: string;
  problemFound: string;
  recorderName: string;
  remark: string;
}

function emptyForm(taskId = ''): RecordFormValues {
  return {
    taskId,
    cleanedAt: today(),
    lengthM: '',
    sludgeVolumeM3: '',
    waterVolumeM3: '',
    personnelCount: '',
    method: '',
    equipment: '',
    weather: '',
    sludgeDisposalSite: '',
    safetyMeasures: '',
    problemFound: '',
    recorderName: '',
    remark: ''
  };
}

function toFormValues(record: CleaningRecord): RecordFormValues {
  return {
    taskId: String(record.taskId),
    cleanedAt: record.cleanedAt ?? '',
    lengthM: String(record.lengthM),
    sludgeVolumeM3: String(record.sludgeVolumeM3),
    waterVolumeM3: String(record.waterVolumeM3),
    personnelCount: String(record.personnelCount),
    method: record.method,
    equipment: record.equipment,
    weather: record.weather,
    sludgeDisposalSite: record.sludgeDisposalSite,
    safetyMeasures: record.safetyMeasures,
    problemFound: record.problemFound,
    recorderName: record.recorderName,
    remark: record.remark
  };
}

function toPayload(values: RecordFormValues): RecordPayload {
  return {
    taskId: Number(values.taskId),
    cleanedAt: values.cleanedAt,
    lengthM: Number(values.lengthM),
    sludgeVolumeM3: Number(values.sludgeVolumeM3),
    waterVolumeM3: values.waterVolumeM3 === '' ? 0 : Number(values.waterVolumeM3),
    personnelCount: Number(values.personnelCount),
    method: values.method as RecordPayload['method'],
    equipment: values.equipment.trim(),
    weather: values.weather as RecordPayload['weather'],
    sludgeDisposalSite: values.sludgeDisposalSite.trim(),
    safetyMeasures: values.safetyMeasures.trim(),
    problemFound: values.problemFound.trim(),
    recorderName: values.recorderName.trim(),
    remark: values.remark.trim()
  };
}

function validate(values: RecordFormValues): FormErrors<RecordFormValues> {
  const errors: FormErrors<RecordFormValues> = {};
  if (!values.taskId) {
    errors.taskId = '请选择关联清淤任务';
  }
  if (!values.cleanedAt) {
    errors.cleanedAt = '清淤日期不能为空';
  } else if (!isDateString(values.cleanedAt)) {
    errors.cleanedAt = '清淤日期格式应为 YYYY-MM-DD';
  } else if (values.cleanedAt > today()) {
    errors.cleanedAt = '清淤日期不能晚于今天';
  }

  const length = Number(values.lengthM);
  if (values.lengthM === '' || Number.isNaN(length) || length <= 0 || length > 100000) {
    errors.lengthM = '清淤长度需大于 0 且不超过 100000（m）';
  }
  const sludge = Number(values.sludgeVolumeM3);
  if (values.sludgeVolumeM3 === '' || Number.isNaN(sludge) || sludge <= 0 || sludge > 100000) {
    errors.sludgeVolumeM3 = '清淤量需大于 0 且不超过 100000（m³）';
  }
  const water = Number(values.waterVolumeM3);
  if (values.waterVolumeM3 === '' || Number.isNaN(water) || water < 0 || water > 100000) {
    errors.waterVolumeM3 = '用水量需在 0 ~ 100000 之间（m³）';
  }
  const personnel = Number(values.personnelCount);
  if (values.personnelCount === '' || !Number.isInteger(personnel) || personnel <= 0 || personnel > 500) {
    errors.personnelCount = '作业人数需为 1 ~ 500 之间的整数';
  }
  if (!values.recorderName.trim()) {
    errors.recorderName = '记录人不能为空';
  }
  return errors;
}

export function RecordFormPage() {
  const params = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();
  const id = Number(params.id ?? '0');
  const isEdit = id > 0;

  const form = useForm<RecordFormValues>(emptyForm(searchParams.get('taskId') ?? ''));
  const [hydrated, setHydrated] = useState(false);

  const detail = useAsync(() => (isEdit ? recordApi.detail(id) : Promise.resolve(null)), [id, isEdit]);
  const tasks = useAsync(() => taskApi.list({ pageSize: 100 }), []);

  useEffect(() => {
    const record = detail.data?.record;
    if (record && !hydrated) {
      form.reset(toFormValues(record));
      setHydrated(true);
    }
  }, [detail.data, hydrated, form]);

  const submit = () => {
    void form.handleSubmit(async () => {
      const payload = toPayload(form.values);
      if (isEdit) {
        await recordApi.update(id, payload);
        toast.success('清淤记录已保存');
        navigate(`/records/${id}`);
      } else {
        const created = await recordApi.create(payload);
        toast.success('清淤记录已录入');
        navigate(`/records/${created.id}`);
      }
    }, validate);
  };

  // 只有待开工 / 清淤中的任务可以录入清淤记录。
  const assignable = (tasks.data?.list ?? []).filter(
    (item) => item.status === 'pending' || item.status === 'in_progress'
  );
  const currentTaskId = Number(form.values.taskId || '0');
  const currentTask = (tasks.data?.list ?? []).find((item) => item.id === currentTaskId);
  const options =
    currentTask && !assignable.some((item) => item.id === currentTask.id) ? [currentTask, ...assignable] : assignable;

  return (
    <form
      className="page"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <PageHeader
        title={isEdit ? '编辑清淤记录' : '录入清淤记录'}
        description="记录单次清淤作业的现场数据；首次录入会自动把任务从「待开工」推进到「清淤中」。"
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

        {tasks.error ? (
          <div className="alert alert-warn">
            <p>任务下拉加载失败：{tasks.error}</p>
          </div>
        ) : null}

        <SectionCard title="作业信息" subtitle="带 * 的字段为必填项">
          <div className="form-grid">
            <FormField
              label="关联清淤任务"
              required
              span={2}
              error={form.errors.taskId}
              hint={isEdit ? '清淤记录不支持更换所属任务，如需调整请删除后重新录入' : `仅列出待开工 / 清淤中的任务，共 ${assignable.length} 条`}
            >
              <select
                className="select"
                value={form.values.taskId}
                disabled={isEdit}
                onChange={(event) => form.setValue('taskId', event.target.value)}
              >
                <option value="">请选择任务</option>
                {options.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.code} · {item.title}
                    {item.status ? `（${optionLabel(enums?.taskStatuses, item.status)}）` : ''}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="清淤日期" required error={form.errors.cleanedAt}>
              <input
                className="input"
                type="date"
                value={form.values.cleanedAt}
                onChange={(event) => form.setValue('cleanedAt', event.target.value)}
              />
            </FormField>
            <FormField label="清淤长度（m）" required error={form.errors.lengthM}>
              <input
                className="input"
                inputMode="decimal"
                value={form.values.lengthM}
                onChange={(event) => form.setValue('lengthM', event.target.value)}
              />
            </FormField>
            <FormField label="清淤量（m³）" required error={form.errors.sludgeVolumeM3}>
              <input
                className="input"
                inputMode="decimal"
                value={form.values.sludgeVolumeM3}
                onChange={(event) => form.setValue('sludgeVolumeM3', event.target.value)}
              />
            </FormField>
            <FormField label="用水量（m³）" required error={form.errors.waterVolumeM3}>
              <input
                className="input"
                inputMode="decimal"
                value={form.values.waterVolumeM3}
                onChange={(event) => form.setValue('waterVolumeM3', event.target.value)}
              />
            </FormField>
            <FormField label="作业人数" required error={form.errors.personnelCount}>
              <input
                className="input"
                inputMode="numeric"
                value={form.values.personnelCount}
                onChange={(event) => form.setValue('personnelCount', event.target.value)}
              />
            </FormField>
            <FormField label="清淤方式" error={form.errors.method}>
              <select
                className="select"
                value={form.values.method}
                onChange={(event) => form.setValue('method', event.target.value)}
              >
                <option value="">未填写</option>
                {(enums?.cleaningMethods ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="天气" error={form.errors.weather}>
              <select
                className="select"
                value={form.values.weather}
                onChange={(event) => form.setValue('weather', event.target.value)}
              >
                <option value="">未填写</option>
                {(enums?.weathers ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="主要设备" error={form.errors.equipment}>
              <input
                className="input"
                value={form.values.equipment}
                placeholder="例如 高压清洗车 + 吸污车"
                onChange={(event) => form.setValue('equipment', event.target.value)}
              />
            </FormField>
            <FormField label="污泥消纳点" error={form.errors.sludgeDisposalSite}>
              <input
                className="input"
                value={form.values.sludgeDisposalSite}
                onChange={(event) => form.setValue('sludgeDisposalSite', event.target.value)}
              />
            </FormField>
            <FormField label="记录人" required error={form.errors.recorderName}>
              <input
                className="input"
                value={form.values.recorderName}
                onChange={(event) => form.setValue('recorderName', event.target.value)}
              />
            </FormField>
            <FormField label="安全措施" span={3} error={form.errors.safetyMeasures}>
              <textarea
                className="textarea"
                value={form.values.safetyMeasures}
                onChange={(event) => form.setValue('safetyMeasures', event.target.value)}
              />
            </FormField>
            <FormField label="发现的问题" span={3} error={form.errors.problemFound}>
              <textarea
                className="textarea"
                value={form.values.problemFound}
                placeholder="例如 局部管段存在错口、树根侵入，已记录待专项处理"
                onChange={(event) => form.setValue('problemFound', event.target.value)}
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
            <button type="submit" className="btn btn-primary" disabled={form.submitting}>
              {form.submitting ? '保存中…' : '保存'}
            </button>
          </div>
        </SectionCard>
      </StateBlock>
    </form>
  );
}
