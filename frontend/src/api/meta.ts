import type { Enums } from '../types/domain';
import { http } from './client';

export const metaApi = {
  enums: () => http.get<Enums>('/meta/enums')
};
