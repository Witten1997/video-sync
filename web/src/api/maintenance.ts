import { http } from '@/utils/request'

export const refreshViewCounts = () => {
  return http.post<{ total: number; updated: number; failed: number; message: string }>('/maintenance/refresh-view-counts')
}
