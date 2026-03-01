<template>
  <div class="min-h-screen flex items-center justify-center bg-background-light">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <div class="w-16 h-16 relative flex items-center justify-center mx-auto mb-4">
          <div class="absolute inset-0 bg-blue-100 dark:bg-blue-900/30 rounded-2xl"></div>
          <svg class="w-9 h-9 text-primary relative z-10" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <path d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" stroke-linecap="round" stroke-linejoin="round"></path>
            <path d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" stroke-linecap="round" stroke-linejoin="round"></path>
            <path d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707" stroke-linecap="round" stroke-linejoin="round"></path>
          </svg>
          <div class="absolute -bottom-1.5 -right-1.5 bg-white dark:bg-sidebar-dark rounded-full p-0.5">
            <svg class="w-4.5 h-4.5 text-blue-600" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" stroke-linecap="round" stroke-linejoin="round"></path>
            </svg>
          </div>
        </div>
        <h1 class="text-2xl font-bold text-slate-800">Video Sync</h1>
      </div>
      <el-card shadow="never" class="!border-slate-200">
        <el-form @submit.prevent="handleLogin" :model="form">
          <el-form-item>
            <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" size="large" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" size="large" show-password @keyup.enter="handleLogin" />
          </el-form-item>
          <el-button type="primary" size="large" class="w-full" :loading="loading" @click="handleLogin">登录</el-button>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const form = ref({ username: '', password: '' })

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await authStore.login(form.value)
    router.push('/')
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false
  }
}
</script>
