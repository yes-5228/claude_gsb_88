import type {
  PageResult,
  PipeSegment,
  SegmentDetail,
  SegmentHistoryItem,
  SegmentOptions,
  SegmentPayload
} from '../types/domain';
import { buildQuery, http } from './client';

export interface SegmentQuery {
  keyword?: string;
  district?: string;
  pipeType?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export const segmentApi = {
  list: (query: SegmentQuery) => http.get<PageResult<PipeSegment>>(`/pipe-segments${buildQuery({ ...query })}`),
  detail: (id: number) => http.get<SegmentDetail>(`/pipe-segments/${id}`),
  history: (id: number) => http.get<SegmentHistoryItem[]>(`/pipe-segments/${id}/cleaning-history`),
  options: (keyword?: string) => http.get<SegmentOptions>(`/pipe-segments/options${buildQuery({ keyword })}`),
  create: (payload: SegmentPayload) => http.post<PipeSegment>('/pipe-segments', payload),
  update: (id: number, payload: SegmentPayload) => http.put<PipeSegment>(`/pipe-segments/${id}`, payload),
  remove: (id: number) => http.del<{ id: number }>(`/pipe-segments/${id}`)
};
