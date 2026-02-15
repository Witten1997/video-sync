<template>
  <div class="flex h-screen overflow-hidden bg-background-light">
    <!-- Sidebar -->
    <aside class="w-64 bg-sidebar-light border-r border-slate-200 flex flex-col z-50 shrink-0">
      <div class="p-6 flex items-center gap-3">
        <div class="w-10 h-10 bg-primary rounded-xl flex items-center justify-center text-white">
          <span class="material-icons-round">play_circle</span>
        </div>
        <span class="text-xl font-bold tracking-tight text-slate-800">Video Sync</span>
      </div>
      <nav class="flex-1 px-4 space-y-1 mt-2 overflow-y-auto">
        <template v-for="route in menuRoutes" :key="route.path">
          <!-- Section header -->
          <div
            v-if="route.meta?.section"
            class="pt-4 pb-2 px-4 uppercase text-[10px] font-bold text-slate-400 tracking-widest"
          >
            {{ route.meta.section }}
          </div>
          <router-link
            :to="route.path"
            :class="[
              'flex items-center gap-3 px-4 py-3 rounded-xl transition-all cursor-pointer no-underline',
              isActive(route.path)
                ? 'bg-blue-50 text-primary font-medium'
                : 'text-slate-600 hover:bg-slate-50'
            ]"
          >
            <span class="material-icons-round text-[20px]">{{ route.meta?.materialIcon || 'circle' }}</span>
            <span class="text-sm">{{ route.meta?.title }}</span>
          </router-link>
        </template>
      </nav>
    </aside>

    <!-- Main content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Header -->
      <header class="h-16 bg-white/80 backdrop-blur-md border-b border-slate-200 sticky top-0 z-40 px-8 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-4 flex-1">
          <h2 class="text-lg font-semibold text-slate-800">{{ currentTitle }}</h2>
        </div>
        <div class="flex items-center gap-4">
          <button
            class="p-2 text-slate-500 hover:text-primary transition-colors rounded-lg hover:bg-slate-100 border-0 outline-none"
            @click="handleRefresh"
          >
            <span class="material-icons-round">refresh</span>
          </button>
          <div class="flex items-center gap-3 pl-4 border-l border-slate-200">
            <div class="text-right">
              <p class="text-sm font-semibold text-slate-800">管理员</p>
            </div>
            <div class="w-9 h-9 bg-primary/10 rounded-full flex items-center justify-center">
              <span class="material-icons-round text-primary text-[20px]">person</span>
            </div>
          </div>
        </div>
      </header>

      <!-- Tabs bar -->
      <TabsBar />

      <!-- Content area -->
      <main class="flex-1 overflow-y-auto bg-background-light">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <keep-alive :include="tabsStore.cachedViews">
              <component :is="Component" :key="route.path" />
            </keep-alive>
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTabsStore } from '@/stores/tabs'
import TabsBar from '@/components/TabsBar.vue'

const route = useRoute()
const router = useRouter()
const tabsStore = useTabsStore()

watch(
  () => route.path,
  () => {
    tabsStore.addTab(route)
  },
  { immediate: true }
)

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

const isActive = (path: string) => {
  return '/' + route.path.split('/')[1] === path
}

const currentTitle = computed(() => {
  return route.meta?.title || '首页'
})

const handleRefresh = () => {
  router.go(0)
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Scrollbar for sidebar */
nav::-webkit-scrollbar {
  width: 4px;
}

nav::-webkit-scrollbar-thumb {
  background: #e2e8f0;
  border-radius: 2px;
}
</style>
