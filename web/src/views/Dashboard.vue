<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #ecf5ff; color: #409eff">
              <el-icon :size="32"><FolderOpened /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_video_sources }}</div>
              <div class="stat-label">视频源数量</div>
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
              <div class="stat-value">{{ stats.total_videos }}</div>
              <div class="stat-label">总视频数</div>
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
              <div class="stat-value">{{ stats.downloaded_videos }}</div>
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
              <div class="stat-value">{{ stats.pending_videos }}</div>
              <div class="stat-label">待下载</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="content-row">
      <el-col :span="12">
        <el-card class="content-card">
          <template #header>
            <div class="card-header">
              <span>当前下载任务</span>
              <el-button text @click="refreshTasks">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <div class="task-list">
            <el-empty v-if="stats.current_tasks.length === 0" description="暂无下载任务" />
            <div v-else>
              <div
                v-for="task in stats.current_tasks"
                :key="task.id"
                class="task-item"
              >
                <div class="task-info">
                  <div class="task-name">{{ task.video_name }}</div>
                  <div class="task-status">
                    <el-tag :type="getTaskStatusType(task.status)" size="small">
                      {{ task.status }}
                    </el-tag>
                  </div>
                </div>
                <el-progress
                  :percentage="task.progress"
                  :status="getProgressStatus(task.status)"
                />
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="content-card">
          <template #header>
            <div class="card-header">
              <span>最近活动</span>
              <el-button text @click="refreshActivities">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <div class="activity-list">
            <el-empty
              v-if="stats.recent_activities.length === 0"
              description="暂无活动记录"
            />
            <el-timeline v-else>
              <el-timeline-item
                v-for="activity in stats.recent_activities"
                :key="activity.id"
                :timestamp="formatTime(activity.created_at)"
                placement="top"
              >
                <div class="activity-message">{{ activity.message }}</div>
              </el-timeline-item>
            </el-timeline>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import { getDashboardStats } from '@/api/dashboard'
import type { DashboardStats } from '@/types'

defineOptions({
  name: 'Dashboard'
})

const stats = ref<DashboardStats>({
  total_video_sources: 0,
  total_videos: 0,
  downloaded_videos: 0,
  pending_videos: 0,
  storage_used: '0 GB',
  recent_activities: [],
  current_tasks: []
})

const loading = ref(false)

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const data = await getDashboardStats()
    stats.value = data
  } catch (error) {
    console.error('加载仪表盘数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 刷新任务列表
const refreshTasks = () => {
  loadData()
}

// 刷新活动列表
const refreshActivities = () => {
  loadData()
}

// 获取任务状态标签类型
const getTaskStatusType = (status: string) => {
  const typeMap: Record<string, any> = {
    pending: 'info',
    downloading: 'primary',
    completed: 'success',
    failed: 'danger'
  }
  return typeMap[status] || 'info'
}

// 获取进度条状态
const getProgressStatus = (status: string) => {
  const statusMap: Record<string, any> = {
    completed: 'success',
    failed: 'exception',
    downloading: undefined
  }
  return statusMap[status]
}

// 格式化时间
const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(() => {
  loadData()

  // 每30秒自动刷新
  setInterval(() => {
    loadData()
  }, 30000)
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

.content-card {
  height: 500px;
}

.content-card :deep(.el-card__body) {
  height: calc(100% - 60px);
  overflow-y: auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.task-list {
  padding: 10px 0;
}

.task-item {
  margin-bottom: 20px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 8px;
}

.task-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.task-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.activity-list {
  padding: 10px 0;
}

.activity-message {
  font-size: 14px;
  color: #606266;
}
</style>
