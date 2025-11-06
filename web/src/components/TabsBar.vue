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
  padding: 0;
  height: 40px;
  display: flex;
  align-items: center;
}

.tabs-wrapper {
  display: flex;
  align-items: center;
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  flex: 1;
}

.tabs-wrapper::-webkit-scrollbar {
  height: 0;
}

.tab-item {
  display: inline-flex;
  align-items: center;
  padding: 0 16px;
  height: 32px;
  margin: 4px 0 4px 4px;
  background: #fafafa;
  border: 1px solid #e8e8e8;
  border-radius: 2px;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
  transition: all 0.3s;
}

.tab-item:hover {
  background: #e6f7ff;
  border-color: #91d5ff;
}

.tab-item.active {
  background: #1890ff;
  border-color: #1890ff;
  color: #fff;
}

.tab-title {
  font-size: 14px;
  margin-right: 8px;
}

.tab-close {
  font-size: 12px;
  transition: all 0.3s;
}

.tab-close:hover {
  font-size: 14px;
  color: #ff4d4f;
}

.tab-item.active .tab-close:hover {
  color: #fff;
}
</style>
