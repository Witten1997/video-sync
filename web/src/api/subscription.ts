import request from '@/utils/request'

export interface FavoriteFolder {
  id: number
  fid: number
  mid: number
  title: string
  cover: string
  media_count: number
  attr: number
  ctime: number
  mtime: number
  subscribed: boolean
}

export interface FollowingUser {
  mid: number
  uname: string
  face: string
  sign: string
  mtime: number
  subscribed: boolean
}

export interface SubscribeRequest {
  id: number
  name?: string
  path?: string
}

// 获取我的收藏夹列表
export function getMyFavorites() {
  return request.get<FavoriteFolder[]>('/subscription/favorites')
}

// 获取我关注的UP主列表
export function getMyFollowings(params?: { pn?: number; ps?: number }) {
  return request.get<{
    list: FollowingUser[]
    total: number
    pn: number
    ps: number
  }>('/subscription/followings', { params })
}

// 订阅收藏夹
export function subscribeFavorite(data: SubscribeRequest) {
  return request.post('/subscription/favorites', data)
}

// 订阅UP主
export function subscribeUpper(data: SubscribeRequest) {
  return request.post('/subscription/uppers', data)
}

// 取消订阅收藏夹
export function unsubscribeFavorite(fid: number) {
  return request.delete(`/subscription/favorites/${fid}`)
}

// 取消订阅UP主
export function unsubscribeUpper(mid: number) {
  return request.delete(`/subscription/uppers/${mid}`)
}
