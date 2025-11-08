<template>
  <div class="dashboard">
    <!-- 第一行：基础统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #ecf5ff; color: #409eff">
              <el-icon :size="32"><FolderOpened /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ dashboardStats.total_sources }}</div>
              <div class="stat-label">视频源总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #f0f9ff; color: #67c23a">
              <el-icon :size="32"><VideoPlay /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ dashboardStats.active_sources }}</div>
              <div class="stat-label">启用的视频源</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #f0f9ff; color: #67c23a">
              <el-icon :size="32"><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ dashboardStats.downloaded_videos }}</div>
              <div class="stat-label">已下载</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #fef0f0; color: #f56c6c">
              <el-icon :size="32"><Clock /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ dashboardStats.pending_videos }}</div>
              <div class="stat-label">待下载</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第二行：存储空间和定时任务 -->
    <el-row :gutter="20" class="content-row">
      <el-col :span="12">
        <el-card class="info-card">
          <template #header>
            <div class="card-header">
              <span>存储空间</span>
              <el-icon><Coin /></el-icon>
            </div>
          </template>
          <div class="storage-info">
            <div class="storage-stats">
              <div class="storage-item">
                <span class="storage-label">总空间：</span>
                <span class="storage-value">{{ formatBytes(dashboardStats.disk_total) }}</span>
              </div>
              <div class="storage-item">
                <span class="storage-label">已用空间：</span>
                <span class="storage-value">{{ formatBytes(dashboardStats.disk_used) }}</span>
              </div>
              <div class="storage-item">
                <span class="storage-label">可用空间：</span>
                <span class="storage-value">{{ formatBytes(dashboardStats.disk_free) }}</span>
              </div>
            </div>
            <el-progress
              :percentage="parseFloat(dashboardStats.disk_used_pct.toFixed(1))"
              :color="getStorageColor(dashboardStats.disk_used_pct)"
              :stroke-width="20"
            />
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="info-card">
          <template #header>
            <div class="card-header">
              <span>定时任务</span>
              <el-icon><Timer /></el-icon>
            </div>
          </template>
          <div class="scheduler-info">
            <el-descriptions :column="1" border>
              <el-descriptions-item label="状态">
                <el-tag :type="schedulerStatus.is_running ? 'success' : 'info'">
                  {{ schedulerStatus.is_running ? '运行中' : '已停止' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="同步间隔">
                {{ schedulerStatus.interval || '-' }}
              </el-descriptions-item>
              <el-descriptions-item label="上次运行">
                {{ formatTime(schedulerStatus.last_run_at) }}
              </el-descriptions-item>
              <el-descriptions-item label="下次运行">
                {{ formatTime(schedulerStatus.next_run_at) }}
              </el-descriptions-item>
            </el-descriptions>
            <div class="scheduler-actions">
              <el-button
                type="primary"
                :loading="triggerLoading"
                @click="handleTriggerSync"
              >
                <el-icon><VideoPlay /></el-icon>
                立即运行
              </el-button>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第三行：系统监控图表 -->
    <el-row :gutter="20" class="content-row">
      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span>CPU 使用率</span>
              <el-text type="info">{{ systemStats.cpu?.percent?.toFixed(1) || 0 }}% ({{ systemStats.cpu?.cores || 0 }} 核)</el-text>
            </div>
          </template>
          <v-chart :option="cpuChartOption" autoresize class="chart" />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span>内存使用率</span>
              <el-text type="info">{{ systemStats.memory?.used_percent?.toFixed(1) || 0 }}% ({{ formatBytes(systemStats.memory?.used) }} / {{ formatBytes(systemStats.memory?.total) }})</el-text>
            </div>
          </template>
          <v-chart :option="memoryChartOption" autoresize class="chart" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 第四行：当前任务和最近活动 -->
    <el-row :gutter="20" class="content-row">
      <el-col :span="12">
        <el-card class="content-card">
          <template #header>
            <div class="card-header">
              <span>当前下载任务</span>
              <el-button text @click="loadDashboardData">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <div class="task-summary">
            <el-tag type="success">运行中: {{ dashboardStats.running_tasks }}</el-tag>
            <el-tag type="info">排队: {{ dashboardStats.pending_tasks }}</el-tag>
            <el-tag>已完成: {{ dashboardStats.completed_tasks }}</el-tag>
            <el-tag type="danger">失败: {{ dashboardStats.failed_tasks }}</el-tag>
          </div>
          <div class="task-progress">
            <el-text v-if="dashboardStats.total_tasks === 0" type="info">暂无下载任务</el-text>
            <el-progress
              v-else
              :percentage="getTaskProgress()"
              :status="dashboardStats.running_tasks > 0 ? undefined : 'success'"
            >
              <span>{{ dashboardStats.completed_tasks }} / {{ dashboardStats.total_tasks }}</span>
            </el-progress>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="content-card">
          <template #header>
            <div class="card-header">
              <span>系统信息</span>
              <el-icon><Monitor /></el-icon>
            </div>
          </template>
          <div class="system-info">
            <el-descriptions :column="2" size="small" border>
              <el-descriptions-item label="Go版本">
                {{ systemStats.go_runtime?.version || '-' }}
              </el-descriptions-item>
              <el-descriptions-item label="协程数">
                {{ systemStats.go_runtime?.goroutines || 0 }}
              </el-descriptions-item>
              <el-descriptions-item label="堆内存">
                {{ formatBytes(systemStats.go_runtime?.heap_alloc) }}
              </el-descriptions-item>
              <el-descriptions-item label="堆对象">
                {{ systemStats.go_runtime?.heap_objects || 0 }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, TitleComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import dayjs from 'dayjs'
import { getDashboardStats } from '@/api/dashboard'
import { getSystemStats } from '@/api/system'
import { getSchedulerStatus, triggerSync } from '@/api/scheduler'

// 注册 ECharts 组件
use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, TitleComponent])

defineOptions({
  name: 'Dashboard'
})

// 数据状态
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

// 图表数据
const cpuData = ref<number[]>([])
const memoryData = ref<number[]>([])
const timeLabels = ref<string[]>([])
const maxDataPoints = 60 // 保存最近60个数据点

// CPU 图表配置
const cpuChartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    formatter: '{b}<br/>{a}: {c}%'
  },
  xAxis: {
    type: 'category',
    data: timeLabels.value,
    boundaryGap: false
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: {
      formatter: '{value}%'
    }
  },
  series: [{
    name: 'CPU',
    type: 'line',
    smooth: true,
    data: cpuData.value,
    symbol: 'none', // 不显示圆点
    areaStyle: {
      color: {
        type: 'linear',
        x: 0,
        y: 0,
        x2: 0,
        y2: 1,
        colorStops: [{
          offset: 0, color: 'rgba(64, 158, 255, 0.5)'
        }, {
          offset: 1, color: 'rgba(64, 158, 255, 0.1)'
        }]
      }
    },
    itemStyle: {
      color: '#409eff'
    }
  }],
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true
  }
}))

// 内存图表配置
const memoryChartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    formatter: '{b}<br/>{a}: {c}%'
  },
  xAxis: {
    type: 'category',
    data: timeLabels.value,
    boundaryGap: false
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: {
      formatter: '{value}%'
    }
  },
  series: [{
    name: '内存',
    type: 'line',
    smooth: true,
    data: memoryData.value,
    symbol: 'none', // 不显示圆点
    areaStyle: {
      color: {
        type: 'linear',
        x: 0,
        y: 0,
        x2: 0,
        y2: 1,
        colorStops: [{
          offset: 0, color: 'rgba(103, 194, 58, 0.5)'
        }, {
          offset: 1, color: 'rgba(103, 194, 58, 0.1)'
        }]
      }
    },
    itemStyle: {
      color: '#67c23a'
    }
  }],
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true
  }
}))

// 加载仪表盘数据
const loadDashboardData = async () => {
  try {
    const data = await getDashboardStats()
    dashboardStats.value = data
  } catch (error) {
    console.error('加载仪表盘数据失败:', error)
  }
}

// 加载系统统计数据
const loadSystemStats = async () => {
  try {
    const data = await getSystemStats()
    systemStats.value = data

    // 更新图表数据
    const now = dayjs().format('HH:mm:ss')
    timeLabels.value.push(now)
    cpuData.value.push(data.cpu?.percent || 0)
    memoryData.value.push(data.memory?.used_percent || 0)

    // 限制数据点数量
    if (timeLabels.value.length > maxDataPoints) {
      timeLabels.value.shift()
      cpuData.value.shift()
      memoryData.value.shift()
    }
  } catch (error) {
    console.error('加载系统统计失败:', error)
  }
}

// 加载调度器状态
const loadSchedulerStatus = async () => {
  try {
    const data = await getSchedulerStatus()
    schedulerStatus.value = data
  } catch (error) {
    console.error('加载调度器状态失败:', error)
  }
}

// 手动触发同步
const handleTriggerSync = async () => {
  try {
    triggerLoading.value = true
    await triggerSync()
    ElMessage.success('同步任务已触发')
    // 刷新调度器状态
    setTimeout(() => {
      loadSchedulerStatus()
    }, 1000)
  } catch (error: any) {
    ElMessage.error(error.message || '触发同步失败')
  } finally {
    triggerLoading.value = false
  }
}

// 格式化字节
const formatBytes = (bytes: number | undefined) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return '-'
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

// 获取存储颜色
const getStorageColor = (percent: number) => {
  if (percent >= 90) return '#f56c6c'
  if (percent >= 70) return '#e6a23c'
  return '#67c23a'
}

// 获取任务进度
const getTaskProgress = () => {
  if (dashboardStats.value.total_tasks === 0) return 0
  return Math.round((dashboardStats.value.completed_tasks / dashboardStats.value.total_tasks) * 100)
}

let refreshTimer: number | null = null
let systemStatsTimer: number | null = null

onMounted(() => {
  // 初始加载
  loadDashboardData()
  loadSchedulerStatus()
  loadSystemStats()

  // 定时刷新仪表盘数据（30秒）
  refreshTimer = window.setInterval(() => {
    loadDashboardData()
    loadSchedulerStatus()
  }, 30000)

  // 定时刷新系统统计（3秒）
  systemStatsTimer = window.setInterval(() => {
    loadSystemStats()
  }, 3000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
  if (systemStatsTimer) {
    clearInterval(systemStatsTimer)
  }
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  height: 120px;
  cursor: pointer;
  transition: all 0.3s;
}

.stat-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 20px;
}

.stat-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #303133;
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.content-row {
  margin-top: 20px;
}

.info-card {
  min-height: 280px;
}

.chart-card {
  height: 380px;
}

.content-card {
  height: 280px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.storage-info {
  padding: 20px 0;
}

.storage-stats {
  margin-bottom: 20px;
}

.storage-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 14px;
}

.storage-label {
  color: #606266;
}

.storage-value {
  font-weight: 500;
  color: #303133;
}

.scheduler-info {
  padding: 10px 0;
}

.scheduler-actions {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}

.chart {
  height: 300px;
  width: 100%;
}

.task-summary {
  display: flex;
  gap: 10px;
  margin-bottom: 15px;
  flex-wrap: wrap;
}

.task-progress {
  padding: 10px 0;
}

.system-info {
  padding: 10px 0;
}
</style>
