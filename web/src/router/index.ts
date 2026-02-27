import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', hidden: true, public: true }
  },
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
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { title: '用户管理', icon: 'User', materialIcon: 'group', section: '系统管理' }
      },
      {
        path: 'config',
        name: 'Config',
        component: () => import('@/views/Config.vue'),
        meta: { title: '配置', icon: 'Setting', materialIcon: 'settings' }
      },
      {
        path: 'maintenance',
        name: 'Maintenance',
        component: () => import('@/views/Maintenance.vue'),
        meta: { title: '维护工具', icon: 'SetUp', materialIcon: 'build' }
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

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('auth_token')
  if (!to.meta?.public && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

export default router
