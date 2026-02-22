import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login as loginApi } from '@/api/user'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('auth_token') || '')
  const username = ref(localStorage.getItem('username') || '')
  const userId = ref(Number(localStorage.getItem('user_id')) || 0)

  const login = async (form: { username: string; password: string }) => {
    const res = await loginApi(form)
    token.value = res.token
    username.value = res.user.username
    userId.value = res.user.id
    localStorage.setItem('auth_token', res.token)
    localStorage.setItem('username', res.user.username)
    localStorage.setItem('user_id', String(res.user.id))
  }

  const logout = () => {
    token.value = ''
    username.value = ''
    userId.value = 0
    localStorage.removeItem('auth_token')
    localStorage.removeItem('username')
    localStorage.removeItem('user_id')
    router.push('/login')
  }

  const isLoggedIn = () => !!token.value

  return { token, username, userId, login, logout, isLoggedIn }
})
