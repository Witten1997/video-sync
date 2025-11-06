import { http } from '@/utils/request'
import type { Video, Page, PageParams, PageResponse } from '@/types'

// 获取视频列表
export const getVideos = (params?: PageParams & {
  source_id?: number
  source_type?: string
  status?: string
  keyword?: string
}) => {
  return http.get<PageResponse<Video>>('/videos', { params })
}

// 获取视频详情
export const getVideo = (id: number) => {
  return http.get<Video>(`/videos/${id}`)
}

// 获取视频的分P列表
export const getVideoPages = (id: number) => {
  return http.get<Page[]>(`/videos/${id}/pages`)
}

// 删除视频
export const deleteVideo = (id: number) => {
  return http.delete(`/videos/${id}`)
}

// 重新下载视频
export const redownloadVideo = (id: number) => {
  return http.post(`/videos/${id}/download`)
}

// 通过URL下载视频
export const downloadVideoByURL = (url: string) => {
  return http.post<{
    task_id: string
    video: Video
    message: string
  }>('/videos/download-by-url', { url })
}
