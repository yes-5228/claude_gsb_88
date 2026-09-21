import type { DistrictStat, Overview, PendingAcceptanceItem, RecentRecordItem } from '../types/domain';
import { buildQuery, http } from './client';

export const dashboardApi = {
  overview: () => http.get<Overview>('/dashboard/overview'),
  districtStats: () => http.get<DistrictStat[]>('/dashboard/district-stats'),
  pendingAcceptance: (limit = 8) =>
    http.get<PendingAcceptanceItem[]>(`/dashboard/pending-acceptance${buildQuery({ limit })}`),
  recentRecords: (limit = 8) => http.get<RecentRecordItem[]>(`/dashboard/recent-records${buildQuery({ limit })}`)
};
