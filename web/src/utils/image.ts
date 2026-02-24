/**
 * 图片URL处理工具函数
 */

/**
 * 获取图片代理URL
 * 用于解决B站图片防盗链问题
 * @param imageUrl 原始图片URL
 * @param thumbnail 是否使用缩略图（默认true，列表页用；详情页传false）
 * @returns 代理后的URL，如果是本地URL则直接返回
 */
export function getProxiedImageUrl(imageUrl: string | undefined | null, thumbnail = true): string {
  // 如果URL为空，返回默认占位图
  if (!imageUrl) {
    return ''
  }

  // 如果是本地URL（以/downloads开头），直接返回
  if (imageUrl.startsWith('/downloads')) {
    return imageUrl
  }

  // 如果是B站的图片URL，使用代理
  if (
    imageUrl.includes('hdslb.com') ||
    imageUrl.includes('biliimg.com')
  ) {
    // 列表页追加缩放参数，避免加载原图导致滚动卡顿
    let proxyUrl = imageUrl
    if (thumbnail && !imageUrl.includes('@')) {
      proxyUrl = `${imageUrl}@480w_270h_1c.webp`
    }
    return `/api/image-proxy?url=${encodeURIComponent(proxyUrl)}`
  }

  // 其他URL直接返回
  return imageUrl
}
