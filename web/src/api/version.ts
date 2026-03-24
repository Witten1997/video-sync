import { http } from '@/utils/request'

export interface VersionInfo {
  current_version: string
  git_tag: string
  build_time: string
  has_update: boolean
  new_version: string
  download_url: string
  changelog: string
  published_at: string
  checked_at: string
}

export interface UpgradeResult {
  message: string
  version: string
}

export const getVersionInfo = () => {
  return http.get<VersionInfo>('/version')
}

export const checkVersion = () => {
  return http.post<VersionInfo>('/version/check')
}

export const doUpgrade = (version: string) => {
  return http.post<UpgradeResult>('/upgrade', { version })
}
