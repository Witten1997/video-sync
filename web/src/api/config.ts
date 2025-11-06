import { http } from '@/utils/request'
import type { Config } from '@/types'

// 获取配置
export const getConfig = () => {
  return http.get<Config>('/config')
}

// 更新配置
export const updateConfig = (data: Partial<Config>) => {
  return http.post<Config>('/config', data)
}

// 验证配置
export const validateConfig = (data: Partial<Config>) => {
  return http.post<{ valid: boolean; errors?: string[] }>('/config/validate', data)
}

// 验证B站认证信息
export const validateBilibiliCredential = () => {
  return http.post<{
    valid: boolean
    message: string
    user_info?: {
      mid: number
      uname: string
      face: string
      sign: string
      level: number
      vip_type: number
      vip_status: number
    }
  }>('/config/validate-credential')
}
