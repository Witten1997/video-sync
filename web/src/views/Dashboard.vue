<template>
  <div class="p-8 space-y-6">
    <!-- Stats cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex items-center gap-4">
        <div class="w-12 h-12 bg-blue-100 text-blue-600 rounded-xl flex items-center justify-center">
          <span class="material-icons-round">folder_open</span>
        </div>
        <div>
          <p class="text-sm text-slate-500 font-medium">视频源总数</p>
          <h3 class="text-2xl font-bold mt-1">{{ dashboardStats.total_sources }}</h3>
        </div>
      </div>
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex items-center gap-4">
        <div class="w-12 h-12 bg-green-100 text-green-600 rounded-xl flex items-center justify-center">
          <span class="material-icons-round">bolt</span>
        </div>
        <div>
          <p class="text-sm text-slate-500 font-medium">启用的源</p>
          <h3 class="text-2xl font-bold mt-1">{{ dashboardStats.active_sources }}</h3>
        </div>
      </div>
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex items-center gap-4">
        <div class="w-12 h-12 bg-purple-100 text-purple-600 rounded-xl flex items-center justify-center">
          <span class="material-icons-round">cloud_download</span>
        </div>
        <div>
          <p class="text-sm text-slate-500 font-medium">已下载</p>
          <h3 class="text-2xl font-bold mt-1">{{ dashboardStats.downloaded_videos }}</h3>
        </div>
      </div>
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm flex items-center gap-4">
        <div class="w-12 h-12 bg-orange-100 text-orange-600 rounded-xl flex items-center justify-center">
          <span class="material-icons-round">schedule</span>
        </div>
        <div>
          <p class="text-sm text-slate-500 font-medium">待下载</p>
          <h3 class="text-2xl font-bold mt-1">{{ dashboardStats.pending_videos }}</h3>
        </div>
      </div>
    </div>

    <!-- Charts row -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- CPU -->
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <h4 class="font-bold text-sm flex items-center gap-2">
            <span class="material-icons-round text-blue-500">memory</span>
            CPU 使用率
          </h4>
          <span class="text-xs font-bold text-blue-500">{{ systemStats.cpu?.percent?.toFixed(1) || 0 }}%</span>
        </div>
        <v-chart :option="cpuChartOption" autoresize style="height: 160px;" />
      </div>

      <!-- Memory -->
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <h4 class="font-bold text-sm flex items-center gap-2">
            <span class="material-icons-round text-purple-500">pie_chart</span>
            内存使用率
          </h4>
          <span class="text-xs font-bold text-purple-500">
            {{ systemStats.memory?.used_percent?.toFixed(1) || 0 }}%
            ({{ formatBytes(systemStats.memory?.used) }} / {{ formatBytes(systemStats.memory?.total) }})
          </span>
        </div>
        <v-chart :option="memoryChartOption" autoresize style="height: 160px;" />
      </div>

      <!-- Storage -->
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <h4 class="font-bold text-sm flex items-center gap-2">
            <span class="material-icons-round text-orange-500">storage</span>
            存储空间
          </h4>
        </div>
        <div class="flex justify-center mb-4">
          <div class="relative h-24 w-24">
            <svg class="h-full w-full" viewBox="0 0 36 36">
              <path class="text-slate-200" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="currentColor" stroke-width="4" />
              <path class="text-orange-500" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="currentColor" :stroke-dasharray="`${dashboardStats.disk_used_pct || 0}, 100`" stroke-linecap="round" stroke-width="4" />
            </svg>
            <div class="absolute inset-0 flex items-center justify-center flex-col">
              <span class="text-lg font-bold">{{ dashboardStats.disk_used_pct?.toFixed(0) || 0 }}%</span>
            </div>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4 text-center">
          <div>
            <p class="text-[10px] text-slate-500 uppercase font-bold tracking-wider">已使用</p>
            <p class="text-sm font-bold">{{ formatBytes(dashboardStats.disk_used) }}</p>
          </div>
          <div>
            <p class="text-[10px] text-slate-500 uppercase font-bold tracking-wider">剩余</p>
            <p class="text-sm font-bold">{{ formatBytes(dashboardStats.disk_free) }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Tasks and Scheduler -->
    <div class="grid grid-cols-1 xl:grid-cols-3 gap-6">
      <!-- Tasks -->
      <div class="bg-white rounded-2xl border border-slate-100 shadow-sm col-span-1 xl:col-span-2 overflow-hidden">
        <div class="p-6 border-b border-slate-100 flex items-center justify-between">
          <h4 class="font-bold text-sm">当前下载任务</h4>
          <div class="flex gap-2">
            <span class="px-2 py-1 bg-green-100 text-green-700 text-[10px] font-bold rounded uppercase">
              运行中: {{ dashboardStats.running_tasks }}
            </span>
            <span class="px-2 py-1 bg-slate-100 text-slate-700 text-[10px] font-bold rounded uppercase">
              排队: {{ dashboardStats.pending_tasks }}
            </span>
            <span class="px-2 py-1 bg-blue-100 text-blue-700 text-[10px] font-bold rounded uppercase">
              完成: {{ dashboardStats.completed_tasks }}
            </span>
            <span class="px-2 py-1 bg-red-100 text-red-700 text-[10px] font-bold rounded uppercase">
              失败: {{ dashboardStats.failed_tasks }}
            </span>
          </div>
        </div>
        <div class="p-6">
          <div v-if="dashboardStats.total_tasks === 0" class="text-center py-8 text-slate-400 text-sm">
            暂无下载任务
          </div>
          <div v-else>
            <el-progress
              :percentage="getTaskProgress()"
              :status="dashboardStats.running_tasks > 0 ? undefined : 'success'"
              :stroke-width="12"
              style="margin-bottom: 8px;"
            />
            <p class="text-xs text-slate-500 text-center">
              {{ dashboardStats.completed_tasks }} / {{ dashboardStats.total_tasks }} 任务完成
            </p>
          </div>
        </div>
      </div>

      <!-- Scheduler -->
      <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm col-span-1">
        <div class="flex items-center justify-between mb-6">
          <h4 class="font-bold text-sm">定时任务</h4>
          <span
            :class="[
              'px-2 py-1 text-[10px] font-bold rounded uppercase',
              schedulerStatus.is_running
                ? 'bg-green-100 text-green-700'
                : 'bg-slate-100 text-slate-500'
            ]"
          >
            {{ schedulerStatus.is_running ? '运行中' : '已停止' }}
          </span>
        </div>
        <div class="space-y-4">
          <div class="flex items-center gap-4 p-4 bg-slate-50 rounded-xl">
            <div class="bg-blue-500/10 text-blue-500 p-2 rounded-lg">
              <span class="material-icons-round text-xl">timer</span>
            </div>
            <div class="flex-1">
              <p class="text-sm font-semibold">同步间隔</p>
              <p class="text-[10px] text-slate-500 uppercase tracking-wider">{{ schedulerStatus.interval || '-' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-4 p-4 bg-slate-50 rounded-xl">
            <div class="bg-green-500/10 text-green-500 p-2 rounded-lg">
              <span class="material-icons-round text-xl">history</span>
            </div>
            <div class="flex-1">
              <p class="text-sm font-semibold">上次运行</p>
              <p class="text-[10px] text-slate-500">{{ formatTime(schedulerStatus.last_run_at) }}</p>
            </div>
          </div>
        </div>
        <div class="flex gap-3 mt-6">
          <button
            :class="[
              'flex-1 py-3 rounded-xl text-sm font-medium transition-colors flex items-center justify-center gap-2 border-0 outline-none',
              schedulerStatus.is_running
                ? 'bg-red-500 text-white hover:bg-red-600 shadow-lg shadow-red-500/20'
                : 'bg-primary text-white hover:bg-blue-600 shadow-lg shadow-primary/20'
            ]"
            :disabled="actionLoading"
            @click="handleToggleScheduler"
          >
            <span class="material-icons-round text-base">{{ schedulerStatus.is_running ? 'stop' : 'play_arrow' }}</span>
            {{ actionLoading ? '处理中...' : (schedulerStatus.is_running ? '停止运行' : '启动调度') }}
          </button>
          <button
            v-if="schedulerStatus.is_running"
            class="py-3 px-4 bg-slate-100 text-slate-700 rounded-xl text-sm font-medium hover:bg-slate-200 transition-colors flex items-center justify-center gap-2 border-0 outline-none"
            :disabled="triggerLoading"
            @click="handleTriggerSync"
          >
            <span class="material-icons-round text-base">sync</span>
            {{ triggerLoading ? '执行中...' : '立即同步' }}
          </button>
        </div>
      </div>
    </div>

    <!-- System Info -->
    <div class="bg-white p-6 rounded-2xl border border-slate-100 shadow-sm">
      <h4 class="font-bold text-sm mb-4 flex items-center gap-2">
        <span class="material-icons-round text-slate-500">info</span>
        系统信息
      </h4>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="p-3 bg-slate-50 rounded-xl text-center">
          <p class="text-[10px] text-slate-500 uppercase font-bold tracking-wider mb-1">Go版本</p>
          <p class="text-sm font-semibold">{{ systemStats.go_runtime?.version || '-' }}</p>
        </div>
        <div class="p-3 bg-slate-50 rounded-xl text-center">
          <p class="text-[10px] text-slate-500 uppercase font-bold tracking-wider mb-1">协程数</p>
          <p class="text-sm font-semibold">{{ systemStats.go_runtime?.goroutines || 0 }}</p>
        </div>
        <div class="p-3 bg-slate-50 rounded-xl text-center">
          <p class="text-[10px] text-slate-500 uppercase font-bold tracking-wider mb-1">堆内存</p>
          <p class="text-sm font-semibold">{{ formatBytes(systemStats.go_runtime?.heap_alloc) }}</p>
        </div>
        <div class="p-3 bg-slate-50 rounded-xl text-center">
          <p class="text-[10px] text-slate-500 uppercase font-bold tracking-wider mb-1">堆对象</p>
          <p class="text-sm font-semibold">{{ systemStats.go_runtime?.heap_objects || 0 }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, onActivated, onDeactivated, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, TitleComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import dayjs from 'dayjs'
import { getDashboardStats } from '@/api/dashboard'
import { getSystemStats } from '@/api/system'
import { getSchedulerStatus, startScheduler, stopScheduler, triggerSync } from '@/api/scheduler'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, TitleComponent])

defineOptions({
  name: 'Dashboard'
})

const dashboardStats = ref<any>({
  total_sources: 0,
  active_sources: 0,
  total_videos: 0,
  downloaded_videos: 0,
  pending_videos: 0,
  total_tasks: 0,
  running_tasks: 0,
  completed_tasks: 0,
  failed_tasks: 0,
  pending_tasks: 0,
  disk_total: 0,
  disk_used: 0,
  disk_free: 0,
  disk_used_pct: 0
})

const systemStats = ref<any>({
  cpu: { percent: 0, cores: 0 },
  memory: { total: 0, used: 0, free: 0, used_percent: 0 },
  go_runtime: {}
})

const schedulerStatus = ref<any>({
  is_running: false,
  interval: '',
  last_run_at: '',
  next_run_at: ''
})

const triggerLoading = ref(false)
const actionLoading = ref(false)

const cpuData = ref<number[]>([])
const memoryData = ref<number[]>([])
const timeLabels = ref<string[]>([])
const maxDataPoints = 60

const cpuChartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    formatter: '{b}<br/>{a}: {c}%'
  },
  xAxis: {
    type: 'category',
    data: timeLabels.value,
    boundaryGap: false,
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: { show: false }
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: { show: false },
    splitLine: { lineStyle: { color: '#f1f5f9' } }
  },
  series: [{
    name: 'CPU',
    type: 'line',
    smooth: true,
    data: cpuData.value,
    symbol: 'none',
    areaStyle: {
      color: {
        type: 'linear',
        x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
          { offset: 1, color: 'rgba(59, 130, 246, 0.02)' }
        ]
      }
    },
    lineStyle: { color: '#3b82f6', width: 2 },
    itemStyle: { color: '#3b82f6' }
  }],
  grid: { left: 0, right: 0, bottom: 0, top: 10, containLabel: false }
}))

const memoryChartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    formatter: '{b}<br/>{a}: {c}%'
  },
  xAxis: {
    type: 'category',
    data: timeLabels.value,
    boundaryGap: false,
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: { show: false }
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: { show: false },
    splitLine: { lineStyle: { color: '#f1f5f9' } }
  },
  series: [{
    name: '内存',
    type: 'line',
    smooth: true,
    data: memoryData.value,
    symbol: 'none',
    areaStyle: {
      color: {
        type: 'linear',
        x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(168, 85, 247, 0.3)' },
          { offset: 1, color: 'rgba(168, 85, 247, 0.02)' }
        ]
      }
    },
    lineStyle: { color: '#a855f7', width: 2 },
    itemStyle: { color: '#a855f7' }
  }],
  grid: { left: 0, right: 0, bottom: 0, top: 10, containLabel: false }
}))

const loadDashboardData = async () => {
  try {
    const data = await getDashboardStats()
    dashboardStats.value = data
  } catch (error) {
    console.error('加载仪表盘数据失败:', error)
  }
}

const loadSystemStats = async () => {
  try {
    const data = await getSystemStats()
    systemStats.value = data

    const now = dayjs().format('HH:mm:ss')
    timeLabels.value.push(now)
    cpuData.value.push(data.cpu?.percent || 0)
    memoryData.value.push(data.memory?.used_percent || 0)

    if (timeLabels.value.length > maxDataPoints) {
      timeLabels.value.shift()
      cpuData.value.shift()
      memoryData.value.shift()
    }
  } catch (error) {
    console.error('加载系统统计失败:', error)
  }
}

const loadSchedulerStatus = async () => {
  try {
    const data = await getSchedulerStatus()
    schedulerStatus.value = data
  } catch (error) {
    console.error('加载调度器状态失败:', error)
  }
}

const handleToggleScheduler = async () => {
  try {
    actionLoading.value = true
    if (schedulerStatus.value.is_running) {
      await stopScheduler()
      ElMessage.success('调度器已停止')
    } else {
      await startScheduler()
      ElMessage.success('调度器已启动')
    }
    await loadSchedulerStatus()
  } catch (error: any) {
    ElMessage.error(error.message || '操作失败')
  } finally {
    actionLoading.value = false
  }
}

const handleTriggerSync = async () => {
  try {
    triggerLoading.value = true
    await triggerSync()
    ElMessage.success('同步任务已触发')
    setTimeout(() => {
      loadSchedulerStatus()
    }, 1000)
  } catch (error: any) {
    ElMessage.error(error.message || '触发同步失败')
  } finally {
    triggerLoading.value = false
  }
}

const formatBytes = (bytes: number | undefined) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const getTaskProgress = () => {
  if (dashboardStats.value.total_tasks === 0) return 0
  return Math.round((dashboardStats.value.completed_tasks / dashboardStats.value.total_tasks) * 100)
}

let refreshTimer: number | null = null
let systemStatsTimer: number | null = null

onMounted(() => {
  loadDashboardData()
  loadSchedulerStatus()
  loadSystemStats()
})

onActivated(() => {
  refreshTimer = window.setInterval(() => {
    loadDashboardData()
    loadSchedulerStatus()
  }, 30000)

  systemStatsTimer = window.setInterval(() => {
    loadSystemStats()
  }, 3000)
})

onDeactivated(() => {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
  if (systemStatsTimer) { clearInterval(systemStatsTimer); systemStatsTimer = null }
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (systemStatsTimer) clearInterval(systemStatsTimer)
})
</script>
