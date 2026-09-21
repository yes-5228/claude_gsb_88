import type { PageResult, TaskDetail, TaskListItem, TaskPayload } from '../types/domain';
import { buildQuery, http } from './client';

export interface TaskQuery {
  keyword?: string;
  status?: string;
  district?: string;
  priority?: string;
  source?: string;
  pipeSegmentId?: number;
  planFrom?: string;
  planTo?: string;
  page?: number;
  pageSize?: number;
}

export const taskApi = {
  list: (query: TaskQuery) => http.get<PageResult<TaskListItem>>(`/cleaning-tasks${buildQuery({ ...query })}`),
  detail: (id: number) => http.get<TaskDetail>(`/cleaning-tasks/${id}`),
  create: (payload: TaskPayload) => http.post<{ id: number }>('/cleaning-tasks', payload),
  update: (id: number, payload: TaskPayload) => http.put<{ id: number }>(`/cleaning-tasks/${id}`, payload),
  remove: (id: number) => http.del<{ id: number }>(`/cleaning-tasks/${id}`),
  start: (id: number) => http.post<{ id: number }>(`/cleaning-tasks/${id}/start`),
  complete: (id: number) => http.post<{ id: number }>(`/cleaning-tasks/${id}/complete`),
  cancel: (id: number, reason: string) => http.post<{ id: number }>(`/cleaning-tasks/${id}/cancel`, { reason })
};
