import { http } from '@/utils/request'

// 系统统计数据类型
export interface SystemStats {
  cpu: {
    percent: number
    cores: number
  }
  memory: {
    total: number
    used: number
    free: number
    used_percent: number
  }
  go_runtime: {
    goroutines: number
    version: string
    heap_alloc: number
    heap_sys: number
    heap_objects: number
    alloc: number
    total_alloc: number
    sys: number
    num_gc: number
  }
  download_manager: {
    running: boolean
    stats: any
  }
  videos?: {
    total: number
    downloaded: number
    pending: number
  }
  tasks?: {
    total: number
    running: number
    completed: number
    failed: number
    pending: number
  }
  timestamp: number
}

// 获取系统统计数据
export const getSystemStats = () => {
  return http.get<SystemStats>('/system/stats')
}

// 系统告警
export interface SystemAlert {
  key: string
  type: string
  title: string
  message: string
  severity: string
  action: string
  created_at: string
  data?: any
}

export const getSystemAlerts = () => {
  return http.get<{ items: SystemAlert[] }>('/system/alerts')
}
