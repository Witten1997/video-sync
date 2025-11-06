import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { RouteLocationNormalized } from 'vue-router'

export interface TabItem {
  path: string
  name: string
  title: string
  closable: boolean
}

export const useTabsStore = defineStore('tabs', () => {
  // 已打开的标签页列表
  const tabs = ref<TabItem[]>([
    {
      path: '/dashboard',
      name: 'Dashboard',
      title: '仪表盘',
      closable: false // 首页不可关闭
    }
  ])

  // 当前激活的标签页路径
  const activeTab = ref('/dashboard')

  // 需要缓存的组件名列表（用于 keep-alive）
  const cachedViews = ref<string[]>(['Dashboard'])

  // 添加标签页
  const addTab = (route: RouteLocationNormalized) => {
    // 跳过隐藏的路由
    if (route.meta?.hidden) {
      return
    }

    const path = route.path
    const name = route.name as string
    const title = route.meta?.title as string

    // 检查标签是否已存在
    const existTab = tabs.value.find(tab => tab.path === path)
    if (!existTab) {
      tabs.value.push({
        path,
        name,
        title,
        closable: path !== '/dashboard' // 首页不可关闭
      })
    }

    // 添加到缓存列表（如果还没有）
    if (name && !cachedViews.value.includes(name)) {
      cachedViews.value.push(name)
    }

    // 设置当前激活的标签
    activeTab.value = path
  }

  // 关闭标签页
  const removeTab = (path: string) => {
    const index = tabs.value.findIndex(tab => tab.path === path)
    if (index === -1) return

    const tab = tabs.value[index]

    // 首页不可关闭
    if (!tab.closable) return

    // 从标签列表中移除
    tabs.value.splice(index, 1)

    // 从缓存列表中移除
    const cacheIndex = cachedViews.value.indexOf(tab.name)
    if (cacheIndex > -1) {
      cachedViews.value.splice(cacheIndex, 1)
    }

    // 如果关闭的是当前激活的标签，需要切换到其他标签
    if (activeTab.value === path) {
      // 优先切换到右边的标签，如果没有则切换到左边
      if (tabs.value.length > 0) {
        const nextTab = tabs.value[Math.min(index, tabs.value.length - 1)]
        activeTab.value = nextTab.path
        return nextTab.path
      }
    }

    return null
  }

  // 关闭其他标签页
  const removeOtherTabs = (path: string) => {
    const keptTabs = tabs.value.filter(tab => tab.path === path || !tab.closable)
    tabs.value = keptTabs

    // 更新缓存列表，只保留当前标签和不可关闭的标签
    cachedViews.value = keptTabs.map(tab => tab.name)

    activeTab.value = path
  }

  // 关闭所有标签页
  const removeAllTabs = () => {
    const keptTabs = tabs.value.filter(tab => !tab.closable)
    tabs.value = keptTabs

    // 更新缓存列表，只保留不可关闭的标签
    cachedViews.value = keptTabs.map(tab => tab.name)

    activeTab.value = '/dashboard'
  }

  // 设置当前激活的标签
  const setActiveTab = (path: string) => {
    activeTab.value = path
  }

  return {
    tabs,
    activeTab,
    cachedViews,
    addTab,
    removeTab,
    removeOtherTabs,
    removeAllTabs,
    setActiveTab
  }
})
