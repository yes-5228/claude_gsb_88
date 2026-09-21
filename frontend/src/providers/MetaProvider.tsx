// 枚举字典在应用启动时拉取一次，各页面通过 useMeta 读取中文标签。
import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { metaApi } from '../api/meta';
import { useAsync } from '../hooks/useAsync';
import type { Enums, Option } from '../types/domain';

interface MetaValue {
  enums: Enums | null;
  loading: boolean;
  error: string;
  reload: () => void;
  options: (key: keyof Enums) => Option[];
}

const MetaContext = createContext<MetaValue | null>(null);

export function MetaProvider({ children }: { children: ReactNode }) {
  const { data, loading, error, reload } = useAsync<Enums>(() => metaApi.enums(), []);

  const value = useMemo<MetaValue>(
    () => ({
      enums: data,
      loading,
      error,
      reload,
      options: (key) => data?.[key] ?? []
    }),
    [data, loading, error, reload]
  );

  return <MetaContext.Provider value={value}>{children}</MetaContext.Provider>;
}

export function useMeta(): MetaValue {
  const value = useContext(MetaContext);
  if (!value) {
    throw new Error('useMeta 必须在 MetaProvider 内部使用');
  }
  return value;
}
