export const qualityLabel = (code?: number): string => {
  switch (code) {
    case 60: return '8K'
    case 50: return '4K'
    case 45: return '1080P60'
    case 40: return '1080P'
    case 30: return '720P'
    case 20: return '480P'
    case 10: return '360P'
    default: return ''
  }
}

export const qualityTagType = (code?: number): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  const q = code || 0
  if (q >= 50) return 'danger'
  if (q >= 45) return 'warning'
  if (q >= 40) return 'success'
  if (q >= 30) return 'primary'
  return 'info'
}

export const orientationLabel = (code?: number): string => {
  if (code === 1) return '横屏'
  if (code === 2) return '竖屏'
  return ''
}
