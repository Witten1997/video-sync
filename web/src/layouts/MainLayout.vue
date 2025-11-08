<template>
  <el-container class="layout-container">
    <el-aside width="250px" class="sidebar">
      <div class="logo">
        <el-icon><VideoPlay /></el-icon>
        <span>Video Sync</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        class="sidebar-menu"
        background-color="transparent"
        text-color="#595959"
        active-text-color="#1890ff"
      >
        <el-menu-item
          v-for="route in menuRoutes"
          :key="route.path"
          :index="route.path"
        >
          <el-icon>
            <component :is="route.meta?.icon" />
          </el-icon>
          <span>{{ route.meta?.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-button text @click="handleRefresh">
            <el-icon><Refresh /></el-icon>
          </el-button>
          <el-dropdown>
            <el-button text>
              <el-icon><User /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>用户信息</el-dropdown-item>
                <el-dropdown-item divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <TabsBar />

      <el-main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <keep-alive :include="tabsStore.cachedViews">
              <component :is="Component" :key="route.path" />
            </keep-alive>
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTabsStore } from '@/stores/tabs'
import TabsBar from '@/components/TabsBar.vue'

const route = useRoute()
const router = useRouter()
const tabsStore = useTabsStore()

// 监听路由变化，自动添加标签页
watch(
  () => route.path,
  () => {
    tabsStore.addTab(route)
  },
  { immediate: true }
)

// 获取菜单路由（排除隐藏的路由）
const menuRoutes = computed(() => {
  const routes = router.getRoutes()
  return routes
    .filter(r => r.path.startsWith('/') && r.meta && !r.meta.hidden && r.path !== '/')
    .map(r => ({
      path: r.path,
      name: r.name,
      meta: r.meta
    }))
})

// 当前激活的菜单
const activeMenu = computed(() => {
  return '/' + route.path.split('/')[1]
})

// 当前页面标题
const currentTitle = computed(() => {
  return route.meta?.title || '首页'
})

// 刷新页面
const handleRefresh = () => {
  router.go(0)
}
</script>

<style scoped>
.layout-container {
  width: 100%;
  height: 100%;
}

.sidebar {
  background: linear-gradient(180deg, #fafbfc 0%, #a6a6cd 100%);
  height: 100%;
  overflow-y: auto;
  box-shadow: 2px 0 8px rgba(0, 21, 41, 0.08);
  position: relative;
  border-right: 1px solid #e8e8e8;
}

.sidebar::before {
  content: '';
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 2px;
  background: linear-gradient(180deg,
    rgba(24, 144, 255, 0) 0%,
    rgba(24, 144, 255, 0.2) 50%,
    rgba(24, 144, 255, 0) 100%
  );
}

/* 自定义滚动条 */
.sidebar::-webkit-scrollbar {
  width: 6px;
}

.sidebar::-webkit-scrollbar-thumb {
  background: #d9d9d9;
  border-radius: 3px;
}

.sidebar::-webkit-scrollbar-thumb:hover {
  background: #bfbfbf;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 64px;
  color: #1890ff;
  font-size: 20px;
  font-weight: bold;
  gap: 10px;
  position: relative;
  margin-bottom: 8px;
  background: linear-gradient(135deg,
    rgba(24, 144, 255, 0.08) 0%,
    rgba(54, 207, 201, 0.08) 100%
  );
}

.logo::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 20px;
  right: 20px;
  height: 2px;
  background: linear-gradient(90deg,
    transparent 0%,
    rgba(24, 144, 255, 0.3) 50%,
    transparent 100%
  );
}

.logo .el-icon {
  font-size: 28px;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  filter: drop-shadow(0 0 6px rgba(24, 144, 255, 0.3));
}

.sidebar-menu {
  border-right: none;
  padding: 8px 12px;
}

/* 自定义菜单项样式 */
.sidebar-menu :deep(.el-menu-item) {
  margin-bottom: 4px;
  border-radius: 8px;
  height: 48px;
  line-height: 48px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  color: #595959;
}

.sidebar-menu :deep(.el-menu-item)::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: linear-gradient(180deg, #1890ff, #36cfc9);
  transform: scaleY(0);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 0 3px 3px 0;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background: #e6f7ff !important;
  color: #1890ff !important;
  transform: translateX(4px);
}

.sidebar-menu :deep(.el-menu-item:hover)::before {
  transform: scaleY(1);
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background: linear-gradient(135deg,
    rgba(24, 144, 255, 0.12) 0%,
    rgba(54, 207, 201, 0.12) 100%
  ) !important;
  color: #1890ff !important;
  font-weight: 500;
  border: 1px solid rgba(24, 144, 255, 0.2);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
}

.sidebar-menu :deep(.el-menu-item.is-active)::before {
  transform: scaleY(1);
}

.sidebar-menu :deep(.el-menu-item .el-icon) {
  font-size: 18px;
  margin-right: 8px;
  transition: all 0.3s;
  color: #8c8c8c;
}

.sidebar-menu :deep(.el-menu-item:hover .el-icon) {
  transform: scale(1.1);
  color: #36cfc9;
}

.sidebar-menu :deep(.el-menu-item.is-active .el-icon) {
  color: #36cfc9;
  filter: drop-shadow(0 0 3px rgba(54, 207, 201, 0.4));
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #f0f0f0;
  background: #fff;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.header-left {
  flex: 1;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 15px;
}

.header-right :deep(.el-button) {
  transition: all 0.3s;
}

.header-right :deep(.el-button:hover) {
  color: #1890ff;
  transform: scale(1.1);
}

.main-content {
  background: #f0f2f5;
  overflow-y: auto;
  padding: 20px;
}

/* 页面切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
