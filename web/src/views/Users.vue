<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <h3 class="text-lg font-semibold text-slate-800">用户管理</h3>
      <el-button type="primary" @click="showAddDialog">
        <span class="material-icons-round text-sm mr-1">add</span>新增用户
      </el-button>
    </div>

    <el-card shadow="never" class="!border-slate-200">
      <el-table :data="users" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column prop="created_at" label="创建时间">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="showEditDialog(row)">编辑</el-button>
            <el-popconfirm title="确定删除此用户？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingUser ? '编辑用户' : '新增用户'" width="400px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password :placeholder="editingUser ? '留空则不修改' : ''" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getUsers, createUser, updateUser, deleteUser } from '@/api/user'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import type { User } from '@/types'

const authStore = useAuthStore()

const users = ref<User[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)
const editingUser = ref<User | null>(null)
const form = ref({ username: '', password: '' })

const formatTime = (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm:ss')

const fetchUsers = async () => {
  loading.value = true
  try {
    users.value = await getUsers()
  } finally {
    loading.value = false
  }
}

const showAddDialog = () => {
  editingUser.value = null
  form.value = { username: '', password: '' }
  dialogVisible.value = true
}

const showEditDialog = (user: User) => {
  editingUser.value = user
  form.value = { username: user.username, password: '' }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.username) {
    ElMessage.warning('请输入用户名')
    return
  }
  if (!editingUser.value && !form.value.password) {
    ElMessage.warning('请输入密码')
    return
  }
  submitting.value = true
  try {
    if (editingUser.value) {
      const isSelf = editingUser.value.username === authStore.username
      const data: any = { username: form.value.username }
      if (form.value.password) data.password = form.value.password
      await updateUser(editingUser.value.id, data)
      ElMessage.success('更新成功')
      if (isSelf) {
        ElMessage.info('当前用户信息已修改，请重新登录')
        setTimeout(() => authStore.logout(), 1000)
        return
      }
    } else {
      await createUser(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchUsers()
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteUser(id)
    ElMessage.success('删除成功')
    fetchUsers()
  } catch {
    // handled by interceptor
  }
}

onMounted(fetchUsers)
</script>
