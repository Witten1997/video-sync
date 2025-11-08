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
  timestamp: number
}

// 获取系统统计数据
export const getSystemStats = () => {
  return http.get<SystemStats>('/system/stats')
}
