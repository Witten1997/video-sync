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
        meta: { title: '仪表盘', icon: 'Odometer' }
      },
      {
        path: 'subscription',
        name: 'Subscription',
        component: () => import('@/views/Subscription.vue'),
        meta: { title: '快捷订阅', icon: 'Star' }
      },
      {
        path: 'video-sources',
        name: 'VideoSources',
        component: () => import('@/views/VideoSources.vue'),
        meta: { title: '视频源管理', icon: 'FolderOpened' }
      },
      {
        path: 'videos',
        name: 'Videos',
        component: () => import('@/views/Videos.vue'),
        meta: { title: '视频列表', icon: 'VideoPlay' }
      },
      {
        path: 'videos/:id',
        name: 'VideoDetail',
        component: () => import('@/views/VideoDetail.vue'),
        meta: { title: '视频详情', hidden: true }
      },
      {
        path: 'tasks',
        name: 'TaskManager',
        component: () => import('@/views/TaskManager.vue'),
        meta: { title: '任务管理', icon: 'List' }
      },
      {
        path: 'sync-logs',
        name: 'SyncLogs',
        component: () => import('@/views/SyncLogs.vue'),
        meta: { title: '同步日志', icon: 'Clock' }
      },
      {
        path: 'config',
        name: 'Config',
        component: () => import('@/views/Config.vue'),
        meta: { title: '配置', icon: 'Setting' }
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/views/Logs.vue'),
        meta: { title: '日志', icon: 'Document' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
