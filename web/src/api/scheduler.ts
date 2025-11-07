import { http } from '@/utils/request'
import type {
  SchedulerStatus,
  SyncLog,
  SyncStats,
  TasksSummary,
  PageParams,
  PageResponse
} from '@/types'

// 获取调度器状态
export const getSchedulerStatus = () => {
  return http.get<SchedulerStatus>('/scheduler/status')
}

// 启动调度器
export const startScheduler = () => {
  return http.post<SchedulerStatus>('/scheduler/start')
}

// 停止调度器
export const stopScheduler = () => {
  return http.post<SchedulerStatus>('/scheduler/stop')
}

// 手动触发同步
export const triggerSync = () => {
  return http.post<{ sync_id: string; started_at: string }>('/scheduler/trigger')
}

// 获取同步日志列表
export const getSyncLogs = (params?: PageParams & {
  trigger_type?: 'auto' | 'manual' | 'all'
  status?: 'running' | 'completed' | 'failed' | 'cancelled' | 'all'
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}) => {
  return http.get<PageResponse<SyncLog>>('/scheduler/logs', { params })
}

// 获取同步日志详情
export const getSyncLog = (id: number) => {
  return http.get<SyncLog>(`/scheduler/logs/${id}`)
}

// 获取同步统计
export const getSyncStats = (period: '1d' | '7d' | '30d' | 'all' = '7d') => {
  return http.get<SyncStats>('/scheduler/stats', { params: { period } })
}

// 获取任务统计
export const getTasksSummary = () => {
  return http.get<TasksSummary>('/scheduler/tasks/summary')
}
