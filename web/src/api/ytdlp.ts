import { http } from '@/utils/request'

// yt-dlp 版本信息
export interface YtdlpVersionInfo {
  current_version: string
  latest_version: string
  has_update: boolean
  update_time?: string
  platform: string
  update_method: string
}

// yt-dlp 更新结果
export interface YtdlpUpdateResult {
  success: boolean
  current_version: string
  old_version: string
  message: string
  output: string
}

// 获取 yt-dlp 版本信息
export const getYtdlpVersionInfo = () => {
  return http.get<YtdlpVersionInfo>('/ytdlp/version')
}

// 更新 yt-dlp
export const updateYtdlpVersion = () => {
  return http.post<YtdlpUpdateResult>('/ytdlp/update', {})
}
