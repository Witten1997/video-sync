<template>
  <div class="subscription-page">
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 我的收藏夹 -->
      <el-tab-pane label="我的收藏夹" name="favorites">
        <div class="header">
          <el-input
            v-model="favoriteSearch"
            placeholder="搜索收藏夹"
            style="width: 300px"
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="header-actions">
            <el-radio-group v-model="favoriteViewMode" size="small">
              <el-radio-button label="list">
                <el-icon><List /></el-icon>
                列表
              </el-radio-button>
              <el-radio-button label="grid">
                <el-icon><Grid /></el-icon>
                卡片
              </el-radio-button>
            </el-radio-group>
            <el-button @click="loadFavorites" :loading="favoritesLoading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>

        <!-- 列表视图 -->
        <el-table
          v-if="favoriteViewMode === 'list'"
          :data="filteredFavorites"
          style="width: 100%; margin-top: 20px"
          v-loading="favoritesLoading"
        >
          <el-table-column prop="title" label="收藏夹名称" min-width="200">
            <template #default="{ row }">
              <div class="folder-info">
                <img :src="row.cover" class="folder-cover" v-if="row.cover" referrerpolicy="no-referrer" />
                <span>{{ row.title }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="media_count" label="视频数量" width="100" />
          <el-table-column prop="fid" label="收藏夹ID" width="150">
            <template #default="{ row }">
              <el-tag type="info" size="small">{{ row.fid }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.subscribed" type="success">已订阅</el-tag>
              <el-tag v-else type="info">未订阅</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="!row.subscribed"
                type="primary"
                size="small"
                @click="handleSubscribeFavorite(row)"
              >
                订阅
              </el-button>
              <el-button
                v-else
                type="danger"
                size="small"
                @click="handleUnsubscribeFavorite(row)"
              >
                取消订阅
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 卡片视图 -->
        <div v-else class="grid-view" v-loading="favoritesLoading">
          <div v-for="item in filteredFavorites" :key="item.id" class="grid-item">
            <el-card :body-style="{ padding: '0px' }" shadow="hover">
              <img :src="item.cover" class="grid-cover" v-if="item.cover" referrerpolicy="no-referrer" />
              <div class="grid-content">
                <div class="grid-title">{{ item.title }}</div>
                <div class="grid-info">
                  <span>{{ item.media_count }} 个视频</span>
                  <el-tag v-if="item.subscribed" type="success" size="small">已订阅</el-tag>
                  <el-tag v-else type="info" size="small">未订阅</el-tag>
                </div>
                <div class="grid-fid">FID: {{ item.fid }}</div>
                <div class="grid-actions">
                  <el-button
                    v-if="!item.subscribed"
                    type="primary"
                    size="small"
                    @click="handleSubscribeFavorite(item)"
                    style="width: 100%"
                  >
                    订阅
                  </el-button>
                  <el-button
                    v-else
                    type="danger"
                    size="small"
                    @click="handleUnsubscribeFavorite(item)"
                    style="width: 100%"
                  >
                    取消订阅
                  </el-button>
                </div>
              </div>
            </el-card>
          </div>
        </div>
      </el-tab-pane>

      <!-- 我关注的UP主 -->
      <el-tab-pane label="我关注的UP主" name="followings">
        <div class="header">
          <el-input
            v-model="followingSearch"
            placeholder="搜索UP主（可搜索所有关注）"
            style="width: 300px"
            clearable
            @input="handleFollowingSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="header-actions">
            <span
              v-if="isSearchingFollowings && followingSearch"
              class="search-tip"
            >
              <el-icon><InfoFilled /></el-icon>
              搜索模式（已加载所有UP主）
            </span>
            <el-radio-group v-model="followingViewMode" size="small">
              <el-radio-button label="list">
                <el-icon><List /></el-icon>
                列表
              </el-radio-button>
              <el-radio-button label="grid">
                <el-icon><Grid /></el-icon>
                卡片
              </el-radio-button>
            </el-radio-group>
            <el-button @click="handleRefreshFollowings" :loading="followingsLoading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>

        <!-- 列表视图 -->
        <el-table
          v-if="followingViewMode === 'list'"
          :data="filteredFollowings"
          style="width: 100%; margin-top: 20px"
          v-loading="followingsLoading"
        >
          <el-table-column prop="uname" label="UP主名称" min-width="200">
            <template #default="{ row }">
              <div class="upper-info">
                <img :src="row.face" class="upper-avatar" alt="头像" referrerpolicy="no-referrer" />
                <div class="upper-details">
                  <div class="upper-name">{{ row.uname }}</div>
                  <div class="upper-sign">{{ row.sign }}</div>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="mid" label="UID" width="120" />
          <el-table-column label="关注时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.mtime) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.subscribed" type="success">已订阅</el-tag>
              <el-tag v-else type="info">未订阅</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="!row.subscribed"
                type="primary"
                size="small"
                @click="handleSubscribeUpper(row)"
              >
                订阅
              </el-button>
              <el-button
                v-else
                type="danger"
                size="small"
                @click="handleUnsubscribeUpper(row)"
              >
                取消订阅
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 卡片视图 -->
        <div v-else class="grid-view" v-loading="followingsLoading">
          <div v-for="item in filteredFollowings" :key="item.mid" class="grid-item">
            <el-card :body-style="{ padding: '12px' }" shadow="hover">
              <div class="upper-card">
                <img :src="item.face" class="upper-card-avatar" alt="头像" referrerpolicy="no-referrer" />
                <div class="upper-card-info">
                  <div class="upper-card-name">{{ item.uname }}</div>
                  <div class="upper-card-uid">UID: {{ item.mid }}</div>
                  <el-tooltip
                    :content="item.sign || '这个UP主很懒，什么都没写'"
                    placement="top"
                    :disabled="!item.sign"
                  >
                    <div class="upper-card-sign">{{ item.sign || '这个UP主很懒，什么都没写' }}</div>
                  </el-tooltip>
                  <div class="upper-card-status">
                    <el-tag v-if="item.subscribed" type="success" size="small">已订阅</el-tag>
                    <el-tag v-else type="info" size="small">未订阅</el-tag>
                  </div>
                  <div class="grid-actions">
                    <el-button
                      v-if="!item.subscribed"
                      type="primary"
                      size="small"
                      @click="handleSubscribeUpper(item)"
                      style="width: 100%"
                    >
                      订阅
                    </el-button>
                    <el-button
                      v-else
                      type="danger"
                      size="small"
                      @click="handleUnsubscribeUpper(item)"
                      style="width: 100%"
                    >
                      取消订阅
                    </el-button>
                  </div>
                </div>
              </div>
            </el-card>
          </div>
        </div>

        <!-- 分页 -->
        <el-pagination
          v-if="followingsTotal > 0 && !isSearchingFollowings"
          style="margin-top: 20px; text-align: right"
          :current-page="followingsPagination.pn"
          :page-size="followingsPagination.ps"
          :total="followingsTotal"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @current-change="handleFollowingPageChange"
          @size-change="handleFollowingSizeChange"
        />

        <!-- 搜索结果提示 -->
        <div v-if="isSearchingFollowings && followingSearch" style="margin-top: 20px; text-align: center; color: #909399;">
          找到 {{ filteredFollowings.length }} 个匹配的UP主
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 订阅对话框 -->
    <el-dialog
      v-model="subscribeDialogVisible"
      :title="subscribeDialogTitle"
      width="500px"
    >
      <el-form :model="subscribeForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="subscribeForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="保存路径">
          <el-input v-model="subscribeForm.path" placeholder="请输入保存路径（可选）">
            <template #append>
              <el-button @click="subscribeForm.path = ''">清空</el-button>
            </template>
          </el-input>
          <div style="font-size: 12px; color: #909399; margin-top: 4px;">
            留空则使用默认保存路径
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="subscribeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubscribeConfirm" :loading="subscribeLoading">
          订阅
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, List, Grid, InfoFilled } from '@element-plus/icons-vue'
import {
  getMyFavorites,
  getMyFollowings,
  subscribeFavorite,
  subscribeUpper,
  unsubscribeFavorite,
  unsubscribeUpper,
  type FavoriteFolder,
  type FollowingUser
} from '@/api/subscription'

// Tab状态
const activeTab = ref('favorites')

// 视图模式
const favoriteViewMode = ref('grid')
const followingViewMode = ref('grid')

// 收藏夹相关
const favorites = ref<FavoriteFolder[]>([])
const favoritesLoading = ref(false)
const favoriteSearch = ref('')

// 关注列表相关
const followings = ref<FollowingUser[]>([])
const followingsLoading = ref(false)
const followingSearch = ref('')
const followingsTotal = ref(0)
const followingsPagination = ref({
  pn: 1,
  ps: 50
})
const isSearchingFollowings = ref(false) // 是否在搜索模式
let searchTimer: NodeJS.Timeout | null = null // 搜索防抖定时器

// 订阅对话框
const subscribeDialogVisible = ref(false)
const subscribeDialogTitle = ref('')
const subscribeLoading = ref(false)
const subscribeForm = ref({
  name: '',
  path: ''
})
const subscribeType = ref<'favorite' | 'upper'>('favorite')
const subscribeTarget = ref<any>(null)

// 过滤后的收藏夹列表
const filteredFavorites = computed(() => {
  if (!favoriteSearch.value) {
    return favorites.value
  }
  return favorites.value.filter(item =>
    item.title.toLowerCase().includes(favoriteSearch.value.toLowerCase())
  )
})

// 过滤后的关注列表（在搜索模式下进行前端过滤）
const filteredFollowings = computed(() => {
  if (!followingSearch.value || !isSearchingFollowings.value) {
    return followings.value
  }
  // 搜索模式下，在所有数据中过滤
  const keyword = followingSearch.value.toLowerCase()
  return followings.value.filter(item =>
    item.uname.toLowerCase().includes(keyword) ||
    item.mid.toString().includes(keyword)
  )
})

// 加载收藏夹列表
const loadFavorites = async () => {
  favoritesLoading.value = true
  try {
    const res = await getMyFavorites()
    // request拦截器已经提取了data.data，直接使用res
    favorites.value = Array.isArray(res) ? res : []
  } catch (error: any) {
    console.error('加载收藏夹失败:', error)
    ElMessage.error(error.response?.data?.message || error.message || '加载收藏夹失败')
  } finally {
    favoritesLoading.value = false
  }
}

// 加载关注列表
const loadFollowings = async (searchMode = false) => {
  followingsLoading.value = true
  try {
    const params: any = searchMode
      ? { all: true }
      : followingsPagination.value

    const res = await getMyFollowings(params)
    // request拦截器已经提取了data.data，直接使用res
    const data = res as any
    followings.value = Array.isArray(data?.list) ? data.list : []
    followingsTotal.value = data?.total || 0
    isSearchingFollowings.value = searchMode
  } catch (error: any) {
    console.error('加载关注列表失败:', error)
    ElMessage.error(error.response?.data?.message || error.message || '加载关注列表失败')
  } finally {
    followingsLoading.value = false
  }
}

// 处理搜索输入（防抖）
const handleFollowingSearch = () => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }

  searchTimer = setTimeout(() => {
    if (followingSearch.value.trim()) {
      // 有搜索内容，进入搜索模式，获取所有数据
      loadFollowings(true)
    } else {
      // 清空搜索，恢复分页模式
      followingsPagination.value.pn = 1
      loadFollowings(false)
    }
  }, 500) // 500ms 防抖
}

// 刷新关注列表
const handleRefreshFollowings = () => {
  // 清空搜索框并恢复分页模式
  followingSearch.value = ''
  isSearchingFollowings.value = false
  followingsPagination.value.pn = 1
  loadFollowings(false)
}

// 订阅收藏夹
const handleSubscribeFavorite = async (row: FavoriteFolder) => {
  subscribeType.value = 'favorite'
  subscribeTarget.value = row
  subscribeDialogTitle.value = '订阅收藏夹'
  subscribeForm.value = {
    name: row.title,
    path: ''
  }
  subscribeDialogVisible.value = true
}

// 取消订阅收藏夹
const handleUnsubscribeFavorite = async (row: FavoriteFolder) => {
  try {
    await ElMessageBox.confirm('确定取消订阅该收藏夹吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await unsubscribeFavorite(row.id)
    ElMessage.success('已取消订阅')
    await loadFavorites()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '取消订阅失败')
    }
  }
}

// 订阅UP主
const handleSubscribeUpper = async (row: FollowingUser) => {
  subscribeType.value = 'upper'
  subscribeTarget.value = row
  subscribeDialogTitle.value = '订阅UP主'
  subscribeForm.value = {
    name: row.uname,
    path: ''
  }
  subscribeDialogVisible.value = true
}

// 取消订阅UP主
const handleUnsubscribeUpper = async (row: FollowingUser) => {
  try {
    await ElMessageBox.confirm('确定取消订阅该UP主吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await unsubscribeUpper(row.mid)
    ElMessage.success('已取消订阅')
    await loadFollowings()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '取消订阅失败')
    }
  }
}

// 确认订阅
const handleSubscribeConfirm = async () => {
  if (!subscribeForm.value.name.trim()) {
    ElMessage.warning('请输入名称')
    return
  }

  subscribeLoading.value = true
  try {
    if (subscribeType.value === 'favorite') {
      await subscribeFavorite({
        id: subscribeTarget.value.id,
        name: subscribeForm.value.name,
        path: subscribeForm.value.path
      })
      ElMessage.success('订阅成功')
      await loadFavorites()
    } else {
      await subscribeUpper({
        id: subscribeTarget.value.mid,
        name: subscribeForm.value.name,
        path: subscribeForm.value.path
      })
      ElMessage.success('订阅成功')
      await loadFollowings()
    }
    subscribeDialogVisible.value = false
  } catch (error: any) {
    ElMessage.error(error.message || '订阅失败')
  } finally {
    subscribeLoading.value = false
  }
}

// 处理关注列表分页
const handleFollowingPageChange = (page: number) => {
  followingsPagination.value.pn = page
  loadFollowings()
}

const handleFollowingSizeChange = (size: number) => {
  followingsPagination.value.ps = size
  followingsPagination.value.pn = 1
  loadFollowings()
}

// 格式化时间
const formatTime = (timestamp: number) => {
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN')
}

// 组件挂载时加载数据
onMounted(() => {
  loadFavorites()
  loadFollowings()
})
</script>

<style scoped>
.subscription-page {
  padding: 32px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.search-tip {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #409eff;
  padding: 4px 8px;
  background: #ecf5ff;
  border-radius: 6px;
}

.folder-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.folder-cover {
  width: 60px;
  height: 40px;
  object-fit: cover;
  border-radius: 8px;
}

.upper-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.upper-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.upper-details {
  flex: 1;
  min-width: 0;
}

.upper-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.upper-sign {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 卡片视图样式 */
.grid-view {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 20px;
  margin-top: 20px;
}

.grid-item {
  height: 100%;
}

.grid-item :deep(.el-card) {
  border-radius: 12px;
  border: 1px solid #f1f5f9;
}

.grid-cover {
  width: 100%;
  height: 120px;
  object-fit: cover;
  display: block;
  border-radius: 12px 12px 0 0;
}

.grid-content {
  padding: 12px;
}

.grid-title {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.grid-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
  color: #64748b;
}

.grid-time {
  font-size: 12px;
  color: #94a3b8;
  margin-bottom: 12px;
}

.grid-fid {
  font-size: 12px;
  color: #94a3b8;
  margin-bottom: 8px;
}

.grid-actions {
  margin-top: 8px;
}

/* UP主卡片样式 */
.upper-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.upper-card-avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  object-fit: cover;
  margin-bottom: 8px;
}

.upper-card-info {
  width: 100%;
}

.upper-card-name {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upper-card-uid {
  font-size: 11px;
  color: #94a3b8;
  margin-bottom: 6px;
}

.upper-card-sign {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: help;
}

.upper-card-status {
  margin-bottom: 8px;
}

.upper-card-time {
  font-size: 11px;
  color: #94a3b8;
  margin-bottom: 8px;
}
</style>
