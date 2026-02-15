<template>
  <div class="h-10 bg-white border-b border-slate-200 px-4 flex items-center shrink-0">
    <div class="flex items-center h-full overflow-x-auto overflow-y-hidden flex-1 gap-1">
      <div
        v-for="tab in tabsStore.tabs"
        :key="tab.path"
        :class="[
          'inline-flex items-center px-3 h-7 rounded-md cursor-pointer select-none whitespace-nowrap transition-all text-xs',
          tab.path === tabsStore.activeTab
            ? 'bg-primary text-white font-medium shadow-sm'
            : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700'
        ]"
        @click="handleTabClick(tab)"
        @contextmenu.prevent="handleContextMenu($event, tab)"
      >
        <span>{{ tab.title }}</span>
        <span
          v-if="tab.closable"
          class="ml-1.5 w-4 h-4 rounded-full flex items-center justify-center hover:bg-black/10 transition-colors"
          @click.stop="handleTabClose(tab)"
        >
          <span class="material-icons-round text-[12px]">close</span>
        </span>
      </div>
    </div>

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
          <el-dropdown-item @click="handleRefresh">刷新</el-dropdown-item>
          <el-dropdown-item v-if="currentContextTab?.closable" @click="handleClose">关闭</el-dropdown-item>
          <el-dropdown-item @click="handleCloseOthers">关闭其他</el-dropdown-item>
          <el-dropdown-item @click="handleCloseAll">关闭所有</el-dropdown-item>
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

const handleTabClick = (tab: TabItem) => {
  router.push(tab.path)
}

const handleTabClose = (tab: TabItem) => {
  const targetPath = tabsStore.removeTab(tab.path)
  if (targetPath) {
    router.push(targetPath)
  }
}

const handleContextMenu = (event: MouseEvent, tab: TabItem) => {
  currentContextTab.value = tab

  if (contextMenuTrigger.value) {
    contextMenuTrigger.value.style.left = event.clientX + 'px'
    contextMenuTrigger.value.style.top = event.clientY + 'px'
    contextMenuTrigger.value.style.visibility = 'visible'

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

const handleRefresh = () => {
  if (currentContextTab.value) {
    router.push(currentContextTab.value.path)
    router.go(0)
  }
}

const handleClose = () => {
  if (currentContextTab.value && currentContextTab.value.closable) {
    handleTabClose(currentContextTab.value)
  }
}

const handleCloseOthers = () => {
  if (currentContextTab.value) {
    tabsStore.removeOtherTabs(currentContextTab.value.path)
    router.push(currentContextTab.value.path)
  }
}

const handleCloseAll = () => {
  tabsStore.removeAllTabs()
  router.push('/dashboard')
}
</script>
