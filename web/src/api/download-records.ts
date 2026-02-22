import { http } from '@/utils/request'
import type { DownloadRecord, PageParams, PageResponse } from '@/types'

export const getDownloadRecords = (params?: PageParams & {
  status?: string
  source_type?: string
  source_id?: string
  sync_log_id?: string
  keyword?: string
}) => {
  return http.get<PageResponse<DownloadRecord>>('/download-records', { params })
}

export const getDownloadRecord = (id: number) => {
  return http.get<DownloadRecord>(`/download-records/${id}`)
}

export const retryDownloadRecord = (id: number) => {
  return http.post<{ task_id: string; record_id: number }>(`/download-records/${id}/retry`)
}

export const deleteDownloadRecord = (id: number) => {
  return http.delete<void>(`/download-records/${id}`)
}

export const batchDeleteDownloadRecords = (ids: number[]) => {
  return http.post<void>('/download-records/batch-delete', { ids })
}
