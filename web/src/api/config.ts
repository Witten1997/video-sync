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

// ==================== 二维码登录 API ====================

// 生成二维码
export const generateQRCode = () => {
  return http.get<{
    url: string
    qrcode_key: string
    expires_in: number
  }>('/auth/qrcode/generate')
}

// 轮询二维码状态
export const pollQRCodeStatus = (qrcodeKey: string) => {
  return http.get<{
    status: number
    message: string
  }>(`/auth/qrcode/poll?qrcode_key=${qrcodeKey}`)
}
