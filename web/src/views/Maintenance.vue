<template>
  <div class="maintenance" style="padding: 32px;">
    <div style="margin-bottom: 24px;">
      <h2 style="margin: 0 0 8px; font-size: 1.25rem; font-weight: 700; color: #1e293b;">维护工具</h2>
      <p style="margin: 0; font-size: 0.875rem; color: #64748b;">系统维护和数据修复工具</p>
    </div>

    <el-card>
      <div style="display: flex; gap: 12px; align-items: center;">
        <el-select v-model="selectedTask" placeholder="选择维护任务" style="width: 280px;">
          <el-option label="刷新播放量" value="refresh_view_counts" />
          <el-option label="检查下载状态" value="repair_download" />
          <el-option label="刷新UP主头像" value="refresh_upper_faces" />
        </el-select>
        <el-button type="primary" :loading="running" :disabled="!selectedTask" @click="handleExecute">
          执行
        </el-button>
      </div>

      <div v-if="resultMessage" style="margin-top: 16px;">
        <el-alert :title="resultMessage" :type="resultType" show-icon :closable="false" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { refreshViewCounts, refreshUpperFaces } from '@/api/maintenance'
import { repairDownloadRecords } from '@/api/download-records'

defineOptions({ name: 'Maintenance' })

const selectedTask = ref('')
const running = ref(false)
const resultMessage = ref('')
const resultType = ref<'success' | 'info' | 'warning' | 'error'>('success')

const handleExecute = async () => {
  if (!selectedTask.value) return
  running.value = true
  resultMessage.value = ''

  try {
    if (selectedTask.value === 'refresh_view_counts') {
      const data = await refreshViewCounts()
      resultMessage.value = data.message + (data.failed > 0 ? `，${data.failed} 个失败` : '')
      resultType.value = data.failed > 0 ? 'warning' : 'success'
    } else if (selectedTask.value === 'repair_download') {
      const data = await repairDownloadRecords()
      if (data.repaired > 0) {
        resultMessage.value = `已修复 ${data.repaired} 条记录`
        resultType.value = 'success'
      } else {
        resultMessage.value = '未发现异常记录'
        resultType.value = 'info'
      }
    } else if (selectedTask.value === 'refresh_upper_faces') {
      const data = await refreshUpperFaces()
      resultMessage.value = data.message
      resultType.value = 'success'
    }
  } catch (error: any) {
    resultMessage.value = error?.response?.data?.message || error?.message || '执行失败'
    resultType.value = 'error'
  } finally {
    running.value = false
  }
}
</script>
