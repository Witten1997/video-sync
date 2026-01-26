import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Video-Sync',
  description: 'B站视频同步工具 for NAS',
  themeConfig: {
    logo: '/logo.png',
    nav: [
      { text: '首页', link: '/' },
      { text: '快速开始', link: '/guide/quick-start' },
      { text: 'GitHub', link: 'https://github.com/Witten1997/video-sync' }
    ],
    sidebar: [
      {
        text: '指南',
        items: [
          { text: '简介', link: '/guide/introduction' },
          { text: '快速开始', link: '/guide/quick-start' },
          { text: '配置说明', link: '/guide/configuration' }
        ]
      },
      {
        text: '功能',
        items: [
          { text: '视频源管理', link: '/features/sources' },
          { text: '下载管理', link: '/features/download' },
          { text: '媒体库集成', link: '/features/media-server' }
        ]
      },
      {
        text: '部署',
        items: [
          { text: 'Docker部署', link: '/deployment/docker' },
          { text: '源码部署', link: '/deployment/source' }
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/Witten1997/video-sync' }
    ]
  }
})
