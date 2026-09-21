// 管段新增 / 编辑表单。
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { segmentApi } from '../../api/pipesegments';
import { FormField } from '../../components/FormField';
import { PageHeader } from '../../components/PageHeader';
import { SectionCard } from '../../components/SectionCard';
import { StateBlock } from '../../components/StateBlock';
import { useToast } from '../../components/Toast';
import { useAsync } from '../../hooks/useAsync';
import { useForm, type FormErrors } from '../../hooks/useForm';
import { useMeta } from '../../providers/MetaProvider';
import type { PipeSegment, SegmentPayload } from '../../types/domain';

interface SegmentFormValues {
  code: string;
  name: string;
  district: string;
  roadName: string;
  pipeType: string;
  material: string;
  diameterMm: string;
  lengthM: string;
  depthM: string;
  startManhole: string;
  endManhole: string;
  buildYear: string;
  ownerUnit: string;
  status: string;
  remark: string;
}

const EMPTY_FORM: SegmentFormValues = {
  code: '',
  name: '',
  district: '',
  roadName: '',
  pipeType: 'rainwater',
  material: '',
  diameterMm: '',
  lengthM: '',
  depthM: '',
  startManhole: '',
  endManhole: '',
  buildYear: '',
  ownerUnit: '',
  status: 'normal',
  remark: ''
};

function toFormValues(segment: PipeSegment): SegmentFormValues {
  return {
    code: segment.code,
    name: segment.name,
    district: segment.district,
    roadName: segment.roadName,
    pipeType: segment.pipeType,
    material: segment.material,
    diameterMm: String(segment.diameterMm),
    lengthM: String(segment.lengthM),
    depthM: String(segment.depthM),
    startManhole: segment.startManhole,
    endManhole: segment.endManhole,
    buildYear: segment.buildYear ? String(segment.buildYear) : '',
    ownerUnit: segment.ownerUnit,
    status: segment.status,
    remark: segment.remark
  };
}

function toPayload(values: SegmentFormValues): SegmentPayload {
  return {
    code: values.code.trim(),
    name: values.name.trim(),
    district: values.district.trim(),
    roadName: values.roadName.trim(),
    pipeType: values.pipeType as SegmentPayload['pipeType'],
    material: values.material.trim(),
    diameterMm: Number(values.diameterMm),
    lengthM: Number(values.lengthM),
    depthM: values.depthM === '' ? 0 : Number(values.depthM),
    startManhole: values.startManhole.trim(),
    endManhole: values.endManhole.trim(),
    buildYear: values.buildYear === '' ? 0 : Number(values.buildYear),
    ownerUnit: values.ownerUnit.trim(),
    status: values.status as SegmentPayload['status'],
    remark: values.remark.trim()
  };
}

function validate(values: SegmentFormValues): FormErrors<SegmentFormValues> {
  const errors: FormErrors<SegmentFormValues> = {};
  if (!values.code.trim()) {
    errors.code = '管段编号不能为空';
  } else if (values.code.trim().length < 2) {
    errors.code = '管段编号至少 2 个字符';
  }
  if (!values.name.trim()) {
    errors.name = '管段名称不能为空';
  }
  if (!values.district.trim()) {
    errors.district = '所属片区不能为空';
  }
  if (!values.pipeType) {
    errors.pipeType = '请选择管段类型';
  }

  const diameter = Number(values.diameterMm);
  if (values.diameterMm === '' || Number.isNaN(diameter) || diameter <= 0 || diameter > 5000) {
    errors.diameterMm = '管径需为 1 ~ 5000 之间的整数（mm）';
  }
  const length = Number(values.lengthM);
  if (values.lengthM === '' || Number.isNaN(length) || length <= 0 || length > 100000) {
    errors.lengthM = '管段长度需大于 0 且不超过 100000（m）';
  }
  const depth = Number(values.depthM);
  if (values.depthM === '' || Number.isNaN(depth) || depth < 0 || depth > 50) {
    errors.depthM = '埋深需在 0 ~ 50 之间（m）';
  }
  if (values.buildYear !== '') {
    const year = Number(values.buildYear);
    if (Number.isNaN(year) || year < 1900 || year > 2100) {
      errors.buildYear = '建设年份需在 1900 ~ 2100 之间';
    }
  }
  return errors;
}

export function SegmentFormPage() {
  const params = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const { enums } = useMeta();
  const id = Number(params.id ?? '0');
  const isEdit = id > 0;

  const form = useForm<SegmentFormValues>(EMPTY_FORM);
  const [hydrated, setHydrated] = useState(false);

  const detail = useAsync(
    () => (isEdit ? segmentApi.detail(id) : Promise.resolve(null)),
    [id, isEdit]
  );

  useEffect(() => {
    const segment = detail.data?.segment;
    if (segment && !hydrated) {
      form.reset(toFormValues(segment));
      setHydrated(true);
    }
  }, [detail.data, hydrated, form]);

  const submit = () => {
    void form.handleSubmit(async () => {
      const payload = toPayload(form.values);
      if (isEdit) {
        const updated = await segmentApi.update(id, payload);
        toast.success(`管段 ${updated.code} 已保存`);
        navigate(`/segments/${id}`);
      } else {
        const created = await segmentApi.create(payload);
        toast.success(`管段 ${created.code} 已创建`);
        navigate(`/segments/${created.id}`);
      }
    }, validate);
  };

  return (
    <form
      className="page"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <PageHeader
        title={isEdit ? '编辑管段' : '新增管段'}
        description="管段编号在系统内唯一；管段是清淤任务、清淤记录与验收记录的业务主体。"
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

        <SectionCard title="基础档案" subtitle="带 * 的字段为必填项">
          <div className="form-grid">
            <FormField label="管段编号" required error={form.errors.code}>
              <input
                className="input"
                value={form.values.code}
                placeholder="例如 PS-2024-001"
                onChange={(event) => form.setValue('code', event.target.value)}
              />
            </FormField>
            <FormField label="管段名称" required error={form.errors.name}>
              <input
                className="input"
                value={form.values.name}
                placeholder="例如 解放路雨水管段"
                onChange={(event) => form.setValue('name', event.target.value)}
              />
            </FormField>
            <FormField label="所属片区" required error={form.errors.district}>
              <input
                className="input"
                value={form.values.district}
                placeholder="例如 城东片区"
                onChange={(event) => form.setValue('district', event.target.value)}
              />
            </FormField>
            <FormField label="所在道路" error={form.errors.roadName}>
              <input
                className="input"
                value={form.values.roadName}
                onChange={(event) => form.setValue('roadName', event.target.value)}
              />
            </FormField>
            <FormField label="管段类型" required error={form.errors.pipeType}>
              <select
                className="select"
                value={form.values.pipeType}
                onChange={(event) => form.setValue('pipeType', event.target.value)}
              >
                {(enums?.pipeTypes ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="管材" error={form.errors.material}>
              <select
                className="select"
                value={form.values.material}
                onChange={(event) => form.setValue('material', event.target.value)}
              >
                <option value="">未填写</option>
                {(enums?.materials ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="管径（mm）" required error={form.errors.diameterMm}>
              <input
                className="input"
                inputMode="numeric"
                value={form.values.diameterMm}
                onChange={(event) => form.setValue('diameterMm', event.target.value)}
              />
            </FormField>
            <FormField label="管段长度（m）" required error={form.errors.lengthM}>
              <input
                className="input"
                inputMode="decimal"
                value={form.values.lengthM}
                onChange={(event) => form.setValue('lengthM', event.target.value)}
              />
            </FormField>
            <FormField label="埋深（m）" required error={form.errors.depthM}>
              <input
                className="input"
                inputMode="decimal"
                value={form.values.depthM}
                onChange={(event) => form.setValue('depthM', event.target.value)}
              />
            </FormField>
            <FormField label="起始检查井" error={form.errors.startManhole}>
              <input
                className="input"
                value={form.values.startManhole}
                onChange={(event) => form.setValue('startManhole', event.target.value)}
              />
            </FormField>
            <FormField label="终点检查井" error={form.errors.endManhole}>
              <input
                className="input"
                value={form.values.endManhole}
                onChange={(event) => form.setValue('endManhole', event.target.value)}
              />
            </FormField>
            <FormField label="建设年份" hint="不填表示未知" error={form.errors.buildYear}>
              <input
                className="input"
                inputMode="numeric"
                value={form.values.buildYear}
                onChange={(event) => form.setValue('buildYear', event.target.value)}
              />
            </FormField>
            <FormField label="权属单位" error={form.errors.ownerUnit}>
              <input
                className="input"
                value={form.values.ownerUnit}
                onChange={(event) => form.setValue('ownerUnit', event.target.value)}
              />
            </FormField>
            <FormField label="运行状态" hint="验收合格后系统会自动置为「正常」" error={form.errors.status}>
              <select
                className="select"
                value={form.values.status}
                onChange={(event) => form.setValue('status', event.target.value)}
              >
                {(enums?.segmentStatuses ?? []).map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
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
