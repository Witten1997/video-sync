import { http } from '@/utils/request'

export const refreshViewCounts = () => {
  return http.post<{ total: number; updated: number; failed: number; message: string }>('/maintenance/refresh-view-counts')
}

export const refreshUpperFaces = () => {
  return http.post<{ total: number; message: string }>('/maintenance/refresh-upper-faces')
}
