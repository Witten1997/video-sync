# Bili-Download Web 前端

基于 Vue 3 + TypeScript + Vite + Element Plus 的 B 站视频下载管理系统前端。

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **TypeScript** - JavaScript 的超集
- **Vite** - 下一代前端构建工具
- **Element Plus** - 基于 Vue 3 的组件库
- **Vue Router** - 官方路由管理器
- **Pinia** - Vue 3 的状态管理库
- **Axios** - HTTP 客户端
- **Day.js** - 轻量级日期处理库

## 功能特性

- 📊 **仪表盘** - 查看系统统计和任务状态
- 📁 **视频源管理** - 管理收藏夹、稍后再看、合集、UP主投稿
- 🎬 **视频列表** - 查看、搜索、筛选视频
- ⚙️ **配置管理** - 系统配置和 B 站认证设置
- 📝 **实时日志** - WebSocket 实时日志推送

## 开发指南

### 安装依赖

```bash
npm install
# 或
pnpm install
# 或
yarn install
```

### 启动开发服务器

```bash
npm run dev
```

访问 http://localhost:3000

### 构建生产版本

```bash
npm run build
```

构建产物将生成在 `dist` 目录。

### 预览生产构建

```bash
npm run preview
```

## 项目结构

```
web/
├── src/
│   ├── api/              # API 调用模块
│   │   ├── dashboard.ts
│   │   ├── video-source.ts
│   │   ├── video.ts
│   │   ├── config.ts
│   │   └── task.ts
│   ├── assets/           # 静态资源
│   ├── components/       # 公共组件
│   ├── layouts/          # 布局组件
│   │   └── MainLayout.vue
│   ├── router/           # 路由配置
│   │   └── index.ts
│   ├── types/            # TypeScript 类型定义
│   │   └── index.ts
│   ├── utils/            # 工具函数
│   │   └── request.ts
│   ├── views/            # 页面组件
│   │   ├── Dashboard.vue
│   │   ├── VideoSources.vue
│   │   ├── Videos.vue
│   │   ├── VideoDetail.vue
│   │   ├── Config.vue
│   │   └── Logs.vue
│   ├── App.vue           # 根组件
│   └── main.ts           # 入口文件
├── index.html            # HTML 模板
├── package.json          # 项目依赖
├── tsconfig.json         # TypeScript 配置
├── vite.config.ts        # Vite 配置
└── README.md            # 项目说明
```

## API 接口

前端通过 Vite 代理与后端 API 通信：

- **仪表盘统计**: `GET /api/dashboard`
- **视频源管理**: `GET/POST/PUT/DELETE /api/video_sources`
- **视频列表**: `GET /api/videos`
- **视频详情**: `GET /api/videos/:id`
- **配置管理**: `GET/POST /api/config`
- **WebSocket 日志**: `WS /ws`

## 开发注意事项

1. **后端服务**: 前端开发时需要后端 API 服务运行在 `http://localhost:8080`
2. **代理配置**: Vite 已配置代理，所有 `/api` 和 `/ws` 请求会转发到后端
3. **认证**: API 请求会自动从 localStorage 读取 `auth_token` 并添加到请求头

## 浏览器支持

- Chrome >= 87
- Firefox >= 78
- Safari >= 14
- Edge >= 88

## 许可证

MIT
