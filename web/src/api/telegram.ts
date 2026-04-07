import { http } from '@/utils/request'
import type { PageResponse, TelegramReconnectResult, TelegramRequestLog, TelegramRuntimeStatus, TelegramTestSendResult } from '@/types'

export const getTelegramStatus = () => {
  return http.get<TelegramRuntimeStatus>('/telegram/status')
}

export const getTelegramRequests = (params?: {
  page?: number
  page_size?: number
  status?: string
  chat_id?: string
  user_id?: string
  task_id?: string
  record_id?: string
  keyword?: string
}) => {
  return http.get<PageResponse<TelegramRequestLog>>('/telegram/requests', { params })
}

export const sendTelegramTestMessage = (data: { chat_id: number; message?: string }) => {
  return http.post<TelegramTestSendResult>('/telegram/test-send', data)
}

export const reconnectTelegram = () => {
  return http.post<TelegramReconnectResult>('/telegram/reconnect')
}
