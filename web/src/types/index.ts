// 视频源类型
export type VideoSourceType = 'favorite' | 'watch_later' | 'collection' | 'submission'

// 视频源接口
export interface VideoSource {
  id: number
  type: VideoSourceType
  name: string
  enabled: boolean
  path: string
  latest_row_at?: string
  created_at: string
  // 特定类型的字段
  f_id?: string // 收藏夹ID
  mid?: string // UP主ID/合集UP主ID
  season_id?: string // 合集ID
  series_id?: string // 系列ID
  collection_type?: string // 合集类型
}

// 视频信息
export interface Video {
  id: number
  bvid: string
  name: string
  intro: string
  cover: string
  tags: string[]
  upper_id: number
  upper_name: string
  upper_face: string
  category: number
  pubtime: string
  favtime: string
  ctime: string
  single_page: boolean
  valid: boolean
  should_download: boolean
  download_status: number
  path: string
  favorite_id?: number
  watch_later_id?: number
  collection_id?: number
  submission_id?: number
  created_at: string
  pages?: Page[]
}

// 分P信息
export interface Page {
  id: number
  video_id: number
  cid: number
  pid: number
  name: string
  duration: number
  width: number
  height: number
  image: string
  download_status: number
  path: string
  created_at: string
}

// 配置信息
export interface Config {
  server: {
    bind_address: string
    auth_token: string
  }
  database: {
    host: string
    port: number
    user: string
    password: string
    dbname: string
    sslmode: string
    max_open_conns: number
    max_idle_conns: number
    conn_max_lifetime: number
  }
  sync: {
    interval: number
    scan_only: boolean
  }
  paths: {
    download_base: string
    upper_path: string
  }
  template: {
    video_name: string
    page_name: string
    time_format: string
  }
  bilibili: {
    credential: {
      sessdata: string
      bili_jct: string
      buvid3: string
      dedeuserid: string
      ac_time_value: string
    }
  }
  quality: {
    max_resolution: string
    codec_priority: string[]
    audio_quality: string
    cdn_sort: boolean
  }
  download: {
    skip_poster: boolean
    skip_video_nfo: boolean
    skip_upper: boolean
    skip_danmaku: boolean
    skip_subtitle: boolean
  }
  danmaku: {
    duration: number
    font_name: string
    font_size: number
    width_ratio: number
    horizontal_gap: number
    lane_size: number
    float_percentage: number
    bottom_percentage: number
    opacity: number
    outline_width: number
    time_offset: number
    bold: boolean
  }
  advanced: {
    concurrent_limit: {
      video: number
      page: number
    }
    rate_limit: {
      duration_ms: number
      limit: number
    }
    nfo_time_type: string
    ytdlp_extra_args: string[]
  }
  logging: {
    level: string
    file: string
    max_size_mb: number
    max_backups: number
    max_age_days: number
  }
}

// 仪表盘统计数据
export interface DashboardStats {
  total_video_sources: number
  total_videos: number
  downloaded_videos: number
  pending_videos: number
  storage_used: string
  recent_activities: Activity[]
  current_tasks: Task[]
}

// 活动记录
export interface Activity {
  id: number
  type: string
  message: string
  created_at: string
}

// 下载任务
export interface Task {
  id: string
  type: 'video' | 'page' | 'collection'
  status: 'pending' | 'queued' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled'
  priority: number
  video: Video
  page?: Page
  output_dir: string
  retry_count: number
  max_retries: number
  error_msg?: string
  created_at: string
  started_at?: string
  completed_at?: string
}

// 调度器状态
export interface SchedulerStatus {
  running: boolean
  last_run_at: string
  next_run_at: string
}

// 分页参数
export interface PageParams {
  page?: number
  page_size?: number
  sort_by?: string
  order?: 'asc' | 'desc'
}

// 分页响应
export interface PageResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

// API 响应
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}
