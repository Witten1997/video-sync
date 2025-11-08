<template>
  <div class="tabs-bar">
    <div class="tabs-wrapper">
      <div
        v-for="tab in tabsStore.tabs"
        :key="tab.path"
        class="tab-item"
        :class="{ active: tab.path === tabsStore.activeTab }"
        @click="handleTabClick(tab)"
        @contextmenu.prevent="handleContextMenu($event, tab)"
      >
        <span class="tab-title">{{ tab.title }}</span>
        <el-icon
          v-if="tab.closable"
          class="tab-close"
          @click.stop="handleTabClose(tab)"
        >
          <Close />
        </el-icon>
      </div>
    </div>

    <!-- 右键菜单 -->
    <el-dropdown
      ref="contextMenuRef"
      trigger="contextmenu"
      :teleported="false"
      placement="bottom-start"
    >
      <div
        ref="contextMenuTrigger"
        style="position: fixed; visibility: hidden"
      ></div>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item @click="handleRefresh">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-dropdown-item>
          <el-dropdown-item
            v-if="currentContextTab?.closable"
            @click="handleClose"
          >
            <el-icon><Close /></el-icon>
            关闭
          </el-dropdown-item>
          <el-dropdown-item @click="handleCloseOthers">
            <el-icon><SemiSelect /></el-icon>
            关闭其他
          </el-dropdown-item>
          <el-dropdown-item @click="handleCloseAll">
            <el-icon><CircleClose /></el-icon>
            关闭所有
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTabsStore } from '@/stores/tabs'
import type { TabItem } from '@/stores/tabs'

const router = useRouter()
const tabsStore = useTabsStore()

const contextMenuRef = ref()
const contextMenuTrigger = ref<HTMLElement>()
const currentContextTab = ref<TabItem | null>(null)

// 点击标签页
const handleTabClick = (tab: TabItem) => {
  router.push(tab.path)
}

// 关闭标签页
const handleTabClose = (tab: TabItem) => {
  const targetPath = tabsStore.removeTab(tab.path)
  if (targetPath) {
    router.push(targetPath)
  }
}

// 右键菜单
const handleContextMenu = (event: MouseEvent, tab: TabItem) => {
  currentContextTab.value = tab

  if (contextMenuTrigger.value) {
    contextMenuTrigger.value.style.left = event.clientX + 'px'
    contextMenuTrigger.value.style.top = event.clientY + 'px'
    contextMenuTrigger.value.style.visibility = 'visible'

    // 触发右键菜单
    const clickEvent = new MouseEvent('contextmenu', {
      bubbles: true,
      cancelable: true,
      clientX: event.clientX,
      clientY: event.clientY
    })
    contextMenuTrigger.value.dispatchEvent(clickEvent)

    setTimeout(() => {
      if (contextMenuTrigger.value) {
        contextMenuTrigger.value.style.visibility = 'hidden'
      }
    }, 100)
  }
}

// 刷新
const handleRefresh = () => {
  if (currentContextTab.value) {
    router.push(currentContextTab.value.path)
    router.go(0)
  }
}

// 关闭
const handleClose = () => {
  if (currentContextTab.value && currentContextTab.value.closable) {
    handleTabClose(currentContextTab.value)
  }
}

// 关闭其他
const handleCloseOthers = () => {
  if (currentContextTab.value) {
    tabsStore.removeOtherTabs(currentContextTab.value.path)
    router.push(currentContextTab.value.path)
  }
}

// 关闭所有
const handleCloseAll = () => {
  tabsStore.removeAllTabs()
  router.push('/dashboard')
}
</script>

<style scoped>
.tabs-bar {
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  padding: 0 12px;
  height: 44px;
  display: flex;
  align-items: center;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.tabs-wrapper {
  display: flex;
  align-items: center;
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  flex: 1;
  gap: 4px;
}

.tabs-wrapper::-webkit-scrollbar {
  height: 3px;
}

.tabs-wrapper::-webkit-scrollbar-thumb {
  background: #d9d9d9;
  border-radius: 3px;
}

.tabs-wrapper::-webkit-scrollbar-thumb:hover {
  background: #bfbfbf;
}

.tab-item {
  position: relative;
  display: inline-flex;
  align-items: center;
  padding: 0 20px;
  height: 32px;
  background: #f5f5f5;
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  font-size: 13px;
  color: #595959;
  border: 1px solid transparent;
  overflow: hidden;
}

.tab-item::before {
  content: '';
  position: absolute;
  left: 0;
  bottom: 0;
  width: 100%;
  height: 2px;
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  transform: scaleX(0);
  transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.tab-item:hover {
  background: #e6f7ff;
  color: #1890ff;
  border-color: #91d5ff;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
}

.tab-item.active {
  background: linear-gradient(135deg, #1890ff 0%, #36cfc9 100%);
  color: #fff;
  font-weight: 500;
  border-color: transparent;
  box-shadow: 0 2px 12px rgba(24, 144, 255, 0.35);
}

.tab-item.active::before {
  transform: scaleX(1);
}

.tab-title {
  font-size: 13px;
  margin-right: 6px;
  letter-spacing: 0.3px;
}

.tab-close {
  font-size: 14px;
  padding: 2px;
  border-radius: 50%;
  transition: all 0.25s;
  opacity: 0.7;
}

.tab-close:hover {
  opacity: 1;
  background: rgba(0, 0, 0, 0.1);
  transform: rotate(90deg);
}

.tab-item.active .tab-close {
  opacity: 0.9;
}

.tab-item.active .tab-close:hover {
  background: rgba(255, 255, 255, 0.2);
  opacity: 1;
}

/* 为非激活标签添加左侧小竖线装饰 */
.tab-item:not(.active)::after {
  content: '';
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 14px;
  background: #1890ff;
  border-radius: 2px;
  opacity: 0;
  transition: opacity 0.25s;
}

.tab-item:not(.active):hover::after {
  opacity: 0.5;
}
</style>
