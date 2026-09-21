// 统一处理「加载中 / 失败 / 成功」三种状态，避免每个页面重复写样板。
import { useCallback, useEffect, useRef, useState } from 'react';
import { toErrorMessage } from '../api/client';

export interface AsyncResult<T> {
  data: T | null;
  loading: boolean;
  error: string;
  reload: () => void;
}

/**
 * 执行一个异步加载函数并跟踪其状态。
 *
 * deps 变化或调用 reload 时重新加载；组件卸载后不再写状态，避免内存泄漏告警。
 */
export function useAsync<T>(loader: () => Promise<T>, deps: readonly unknown[] = []): AsyncResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tick, setTick] = useState(0);

  const loaderRef = useRef(loader);
  loaderRef.current = loader;

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setError('');
    loaderRef.current().then(
      (result) => {
        if (!alive) {
          return;
        }
        setData(result);
        setLoading(false);
      },
      (cause: unknown) => {
        if (!alive) {
          return;
        }
        setData(null);
        setError(toErrorMessage(cause));
        setLoading(false);
      }
    );
    return () => {
      alive = false;
    };
  }, [...deps, tick]);

  const reload = useCallback(() => {
    setTick((value) => value + 1);
  }, []);

  return { data, loading, error, reload };
}
