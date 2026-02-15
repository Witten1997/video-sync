import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '控制台', icon: 'Odometer', materialIcon: 'dashboard' }
      },
      {
        path: 'subscription',
        name: 'Subscription',
        component: () => import('@/views/Subscription.vue'),
        meta: { title: '快捷订阅', icon: 'Star', materialIcon: 'star' }
      },
      {
        path: 'video-sources',
        name: 'VideoSources',
        component: () => import('@/views/VideoSources.vue'),
        meta: { title: '视频源管理', icon: 'FolderOpened', materialIcon: 'rss_feed' }
      },
      {
        path: 'videos',
        name: 'Videos',
        component: () => import('@/views/Videos.vue'),
        meta: { title: '视频列表', icon: 'VideoPlay', materialIcon: 'video_library' }
      },
      {
        path: 'videos/:id',
        name: 'VideoDetail',
        component: () => import('@/views/VideoDetail.vue'),
        meta: { title: '视频详情', hidden: true }
      },
      {
        path: 'download-records',
        name: 'DownloadRecords',
        component: () => import('@/views/DownloadRecords.vue'),
        meta: { title: '下载管理', icon: 'Download', materialIcon: 'download' }
      },
      {
        path: 'tasks',
        name: 'TaskManager',
        component: () => import('@/views/TaskManager.vue'),
        meta: { title: '任务管理', icon: 'List', materialIcon: 'assignment' }
      },
      {
        path: 'sync-logs',
        name: 'SyncLogs',
        component: () => import('@/views/SyncLogs.vue'),
        meta: { title: '同步日志', icon: 'Clock', materialIcon: 'sync', section: '系统管理' }
      },
      {
        path: 'config',
        name: 'Config',
        component: () => import('@/views/Config.vue'),
        meta: { title: '配置', icon: 'Setting', materialIcon: 'settings' }
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/views/Logs.vue'),
        meta: { title: '日志', icon: 'Document', materialIcon: 'terminal' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
