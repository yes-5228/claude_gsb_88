// 表单状态管理：值、字段错误、提交中与后端错误提示集中在一处。
import { useCallback, useState } from 'react';
import { toErrorMessage } from '../api/client';

export type FormErrors<T> = Partial<Record<keyof T, string>>;

export interface FormController<T extends object> {
  values: T;
  errors: FormErrors<T>;
  setValue: <K extends keyof T>(key: K, value: T[K]) => void;
  reset: (next: T) => void;
  submitting: boolean;
  serverError: string;
  handleSubmit: (submit: () => Promise<void>, validate?: (values: T) => FormErrors<T>) => Promise<void>;
}

export function useForm<T extends object>(initial: T): FormController<T> {
  const [values, setValues] = useState<T>(initial);
  const [errors, setErrors] = useState<FormErrors<T>>({});
  const [submitting, setSubmitting] = useState(false);
  const [serverError, setServerError] = useState('');

  const setValue = useCallback(<K extends keyof T>(key: K, value: T[K]) => {
    setValues((prev) => ({ ...prev, [key]: value }) as T);
  }, []);

  const reset = useCallback((next: T) => {
    setValues(next);
    setErrors({});
    setServerError('');
  }, []);

  const handleSubmit = useCallback(
    async (submit: () => Promise<void>, validate?: (values: T) => FormErrors<T>) => {
      setServerError('');
      const found = validate ? validate(values) : {};
      setErrors(found);
      if (Object.keys(found).length > 0) {
        return;
      }
      setSubmitting(true);
      try {
        await submit();
      } catch (cause: unknown) {
        setServerError(toErrorMessage(cause));
      } finally {
        setSubmitting(false);
      }
    },
    [values]
  );

  return { values, errors, setValue, reset, submitting, serverError, handleSubmit };
}
