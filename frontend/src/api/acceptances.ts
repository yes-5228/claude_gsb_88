import type {
  AcceptanceDetail,
  AcceptanceListItem,
  AcceptancePayload,
  PageResult,
  RectifyPayload
} from '../types/domain';
import { buildQuery, http } from './client';

export interface AcceptanceQuery {
  keyword?: string;
  taskId?: number;
  segmentId?: number;
  result?: string;
  inspectorName?: string;
  dateFrom?: string;
  dateTo?: string;
  pendingRectify?: boolean;
  page?: number;
  pageSize?: number;
}

export const acceptanceApi = {
  list: (query: AcceptanceQuery) =>
    http.get<PageResult<AcceptanceListItem>>(`/acceptances${buildQuery({ ...query })}`),
  detail: (id: number) => http.get<AcceptanceDetail>(`/acceptances/${id}`),
  create: (payload: AcceptancePayload) => http.post<{ id: number }>('/acceptances', payload),
  rectify: (id: number, payload: RectifyPayload) => http.post<{ id: number }>(`/acceptances/${id}/rectify`, payload),
  remove: (id: number) => http.del<{ id: number }>(`/acceptances/${id}`)
};
