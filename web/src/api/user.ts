import { http } from '@/utils/request'
import type { User } from '@/types'

export const login = (data: { username: string; password: string }) => {
  return http.post<{ token: string; user: { id: number; username: string } }>('/auth/login', data)
}

export const getCurrentUser = () => {
  return http.get<User>('/users/me')
}

export const getUsers = () => {
  return http.get<User[]>('/users')
}

export const createUser = (data: { username: string; password: string }) => {
  return http.post<User>('/users', data)
}

export const updateUser = (id: number, data: { username?: string; password?: string }) => {
  return http.put<User>(`/users/${id}`, data)
}

export const deleteUser = (id: number) => {
  return http.delete(`/users/${id}`)
}

export const changePassword = (data: { old_password: string; new_password: string }) => {
  return http.put('/users/me/password', data)
}
