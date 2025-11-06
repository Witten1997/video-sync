import { http } from '@/utils/request'
import type { DashboardStats } from '@/types'

// 获取仪表盘统计数据
export const getDashboardStats = () => {
  return http.get<DashboardStats>('/dashboard')
}
