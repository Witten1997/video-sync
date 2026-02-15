<template>
  <div class="video-sources">
    <div class="toolbar">
      <el-button type="primary" @click="showAddDialog">
        <el-icon><Plus /></el-icon>
        添加视频源
      </el-button>
      <el-button @click="loadData">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
      <el-button
        v-if="selectedSources.length > 0"
        type="primary"
        @click="handleBatchScan"
      >
        <el-icon><Search /></el-icon>
        批量扫描 ({{ selectedSources.length }})
      </el-button>
      <el-button
        v-if="selectedSources.length > 0"
        type="danger"
        @click="handleBatchDelete"
      >
        <el-icon><Delete /></el-icon>
        批量删除 ({{ selectedSources.length }})
      </el-button>
    </div>

    <!-- 筛选条件 -->
    <div class="filter-bar">
      <el-input
        v-model="filters.name"
        placeholder="搜索视频源名称"
        style="width: 250px"
        clearable
        @input="handleFilter"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>

      <el-select
        v-model="filters.type"
        placeholder="选择类型"
        style="width: 150px"
        clearable
        @change="handleFilter"
      >
        <el-option label="全部类型" value="" />
        <el-option label="收藏夹" value="favorite" />
        <el-option label="稍后再看" value="watch_later" />
        <el-option label="合集" value="collection" />
        <el-option label="UP主投稿" value="submission" />
      </el-select>

      <el-select
        v-model="filters.enabled"
        placeholder="选择状态"
        style="width: 150px"
        clearable
        @change="handleFilter"
      >
        <el-option label="全部状态" value="" />
        <el-option label="已启用" :value="true" />
        <el-option label="已禁用" :value="false" />
      </el-select>

      <el-tag v-if="hasActiveFilters" type="info">
        已筛选 {{ filteredVideoSources.length }} / {{ videoSources.length }} 条
      </el-tag>
    </div>

    <el-table
      v-loading="loading"
      :data="filteredVideoSources"
      border
      stripe
      style="width: 100%"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="源ID" width="150">
        <template #default="{ row }">
          <span>{{ getSourceId(row) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="名称" min-width="200" />
      <el-table-column label="类型" width="120">
        <template #default="{ row }">
          <el-tag :type="getSourceTypeColor(row.type)">
            {{ getSourceTypeName(row.type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="path" label="保存路径" min-width="200" show-overflow-tooltip />
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-switch
            v-model="row.enabled"
            @change="handleToggle(row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="最后扫描" width="180">
        <template #default="{ row }">
          {{ row.latest_row_at ? formatTime(row.latest_row_at) : '未扫描' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" size="small" @click="handleScan(row)">
            <el-icon><Search /></el-icon>
            扫描
          </el-button>
          <el-button text type="primary" size="small" @click="handleEdit(row)">
            <el-icon><Edit /></el-icon>
            编辑
          </el-button>
          <el-button text type="danger" size="small" @click="handleDelete(row)">
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑视频源' : '添加视频源'"
      width="600px"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="视频源类型" prop="type">
          <el-select
            v-model="formData.type"
            placeholder="请选择视频源类型"
            :disabled="isEdit"
          >
            <el-option label="收藏夹" value="favorite" />
            <el-option label="稍后再看" value="watch_later" />
            <el-option label="合集" value="collection" />
            <el-option label="UP主投稿" value="submission" />
          </el-select>
        </el-form-item>

        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入名称" />
        </el-form-item>

        <el-form-item label="保存路径" prop="path">
          <el-input v-model="formData.path" placeholder="请输入保存路径" />
        </el-form-item>

        <!-- 收藏夹特有字段 -->
        <template v-if="formData.type === 'favorite'">
          <el-form-item label="收藏夹ID" prop="f_id">
            <el-input v-model="formData.f_id" placeholder="请输入收藏夹ID" />
          </el-form-item>
        </template>

        <!-- 合集特有字段 -->
        <template v-if="formData.type === 'collection'">
          <el-form-item label="UP主ID" prop="mid">
            <el-input v-model="formData.mid" placeholder="请输入UP主ID" />
          </el-form-item>
          <el-form-item label="合集类型" prop="collection_type">
            <el-radio-group v-model="formData.collection_type">
              <el-radio value="season">合集</el-radio>
              <el-radio value="series">系列</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item
            v-if="formData.collection_type === 'season'"
            label="合集ID"
            prop="season_id"
          >
            <el-input v-model="formData.season_id" placeholder="请输入合集ID" />
          </el-form-item>
          <el-form-item
            v-if="formData.collection_type === 'series'"
            label="系列ID"
            prop="series_id"
          >
            <el-input v-model="formData.series_id" placeholder="请输入系列ID" />
          </el-form-item>
        </template>

        <!-- UP主投稿特有字段 -->
        <template v-if="formData.type === 'submission'">
          <el-form-item label="UP主ID" prop="mid">
            <el-input v-model="formData.mid" placeholder="请输入UP主ID" />
          </el-form-item>
        </template>

        <el-form-item label="启用">
          <el-switch v-model="formData.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Search, Plus, Edit, Delete, Refresh } from '@element-plus/icons-vue'

defineOptions({
  name: 'VideoSources'
})
import {
  getVideoSources,
  createVideoSource,
  updateVideoSource,
  deleteVideoSource,
  toggleVideoSource,
  scanVideoSource
} from '@/api/video-source'
import type { VideoSource } from '@/types'
import dayjs from 'dayjs'

const loading = ref(false)
const videoSources = ref<VideoSource[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const selectedSources = ref<VideoSource[]>([])

// 筛选条件
const filters = ref({
  name: '',
  type: '',
  enabled: '' as boolean | ''
})

// 过滤后的视频源列表
const filteredVideoSources = computed(() => {
  let result = videoSources.value

  // 按名称过滤
  if (filters.value.name) {
    const keyword = filters.value.name.toLowerCase()
    result = result.filter(item => item.name.toLowerCase().includes(keyword))
  }

  // 按类型过滤
  if (filters.value.type) {
    result = result.filter(item => item.type === filters.value.type)
  }

  // 按状态过滤
  if (filters.value.enabled !== '') {
    result = result.filter(item => item.enabled === filters.value.enabled)
  }

  return result
})

// 是否有活动的筛选条件
const hasActiveFilters = computed(() => {
  return filters.value.name !== '' ||
         filters.value.type !== '' ||
         filters.value.enabled !== ''
})

// 处理筛选
const handleFilter = () => {
  // 筛选逻辑由 computed 自动处理
}

const formData = ref<Partial<VideoSource>>({
  type: 'favorite',
  name: '',
  path: '',
  enabled: true,
  f_id: '',
  mid: '',
  season_id: '',
  series_id: '',
  collection_type: 'season'
})

const formRules: FormRules = {
  type: [{ required: true, message: '请选择视频源类型', trigger: 'change' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  path: [{ required: true, message: '请输入保存路径', trigger: 'blur' }]
}

// 加载视频源列表
const loadData = async () => {
  loading.value = true
  try {
    const result = await getVideoSources()
    videoSources.value = result.items || []
  } catch (error) {
    console.error('加载视频源列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 显示添加对话框
const showAddDialog = () => {
  isEdit.value = false
  formData.value = {
    type: 'favorite',
    name: '',
    path: '',
    enabled: true,
    f_id: '',
    mid: '',
    season_id: '',
    series_id: '',
    collection_type: 'season'
  }
  dialogVisible.value = true
}

// 编辑视频源
const handleEdit = (row: VideoSource) => {
  isEdit.value = true
  formData.value = { ...row }
  dialogVisible.value = true
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    try {
      if (isEdit.value) {
        await updateVideoSource(formData.value.id!, formData.value, formData.value.type!)
        ElMessage.success('更新成功')
      } else {
        await createVideoSource(formData.value)
        ElMessage.success('添加成功')
      }
      dialogVisible.value = false
      loadData()
    } catch (error) {
      console.error('操作失败:', error)
    }
  })
}

// 切换启用状态
const handleToggle = async (row: VideoSource) => {
  try {
    await toggleVideoSource(row.id, row.enabled, row.type)
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch (error) {
    row.enabled = !row.enabled
    console.error('切换状态失败:', error)
  }
}

// 扫描视频源
const handleScan = async (row: VideoSource) => {
  try {
    await scanVideoSource(row.id, row.type)
    ElMessage.success('扫描任务已启动')
  } catch (error) {
    console.error('扫描失败:', error)
  }
}

// 删除视频源
const handleDelete = async (row: VideoSource) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除视频源 "${row.name}" 吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteVideoSource(row.id, row.type)
    ElMessage.success('删除成功')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
    }
  }
}

// 获取视频源类型名称
const getSourceTypeName = (type: string) => {
  const typeMap: Record<string, string> = {
    favorite: '收藏夹',
    watch_later: '稍后再看',
    collection: '合集',
    submission: 'UP主投稿'
  }
  return typeMap[type] || type
}

// 获取视频源类型颜色
const getSourceTypeColor = (type: string) => {
  const colorMap: Record<string, string> = {
    favorite: 'primary',
    watch_later: 'success',
    collection: 'warning',
    submission: 'danger'
  }
  return colorMap[type] || ''
}

// 获取源ID
const getSourceId = (row: VideoSource) => {
  switch (row.type) {
    case 'favorite':
      return `FID: ${row.f_id || '-'}`
    case 'collection':
      return `CID: ${row.cid || '-'}`
    case 'submission':
      return `UID: ${row.mid || row.upper_id || '-'}`
    case 'watch_later':
      return '-'
    default:
      return '-'
  }
}

// 格式化时间
const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

// 处理选择变化
const handleSelectionChange = (selection: VideoSource[]) => {
  selectedSources.value = selection
}

// 批量扫描
const handleBatchScan = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要扫描选中的 ${selectedSources.value.length} 个视频源吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info'
      }
    )

    let successCount = 0
    for (const source of selectedSources.value) {
      try {
        await scanVideoSource(source.id, source.type)
        successCount++
      } catch (error) {
        console.error(`扫描视频源 ${source.name} 失败:`, error)
      }
    }
    ElMessage.success(`已启动 ${successCount} 个扫描任务`)
    selectedSources.value = []
  } catch (error) {
    if (error !== 'cancel') {
      console.error('批量扫描失败:', error)
    }
  }
}

// 批量删除
const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedSources.value.length} 个视频源吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    let successCount = 0
    for (const source of selectedSources.value) {
      try {
        await deleteVideoSource(source.id, source.type)
        successCount++
      } catch (error) {
        console.error(`删除视频源 ${source.name} 失败:`, error)
      }
    }
    ElMessage.success(`已删除 ${successCount} 个视频源`)
    selectedSources.value = []
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('批量删除失败:', error)
    }
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.video-sources {
  padding: 32px;
}

.toolbar {
  margin-bottom: 24px;
  display: flex;
  gap: 12px;
  align-items: center;
}

.filter-bar {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 24px;
  padding: 16px 20px;
  background: #ffffff;
  border: 1px solid #f1f5f9;
  border-radius: 12px;
  flex-wrap: wrap;
}
</style>
