import { http } from '@/utils/request'
import type { VideoSource, PageParams, PageResponse } from '@/types'

// 获取视频源列表
export const getVideoSources = (params?: PageParams) => {
  return http.get<PageResponse<VideoSource>>('/video_sources', { params })
}

// 获取单个视频源
export const getVideoSource = (id: number, type: string) => {
  return http.get<VideoSource>(`/video_sources/${id}?type=${type}`)
}

// 创建视频源
export const createVideoSource = (data: Partial<VideoSource>) => {
  return http.post<VideoSource>('/video_sources', data)
}

// 更新视频源
export const updateVideoSource = (id: number, data: Partial<VideoSource>, type: string) => {
  return http.put<VideoSource>(`/video_sources/${id}?type=${type}`, data)
}

// 删除视频源
export const deleteVideoSource = (id: number, type: string) => {
  return http.delete(`/video_sources/${id}?type=${type}`)
}

// 启用/禁用视频源
export const toggleVideoSource = (id: number, enabled: boolean, type: string) => {
  return http.put(`/video_sources/${id}/enable?type=${type}`, { enabled })
}

// 手动触发扫描
export const scanVideoSource = (id: number, type: string) => {
  return http.post(`/video_sources/${id}/scan?type=${type}`)
}
