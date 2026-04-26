import heic2any from 'heic2any'

const objectUrls = new Set<string>()
const resolvedUrlCache = new Map<string, Promise<string>>()

export function isHeicLikeUrl(url: string): boolean {
  return /\.(heic|heif)(?:$|[?#])/i.test(url)
}

export async function decodeHeicToObjectUrl(url: string): Promise<string> {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`failed to fetch image: ${response.status}`)
  }

  const inputBlob = await response.blob()
  const output = await heic2any({
    blob: inputBlob,
    toType: 'image/jpeg',
    quality: 0.92
  })

  const outputBlob = Array.isArray(output) ? output[0] : output
  return URL.createObjectURL(outputBlob)
}

export async function resolveHeicImageUrl(url: string): Promise<string> {
  if (!url || !isHeicLikeUrl(url)) {
    return url
  }

  let task = resolvedUrlCache.get(url)
  if (!task) {
    task = decodeHeicToObjectUrl(url).then((objectUrl) => {
      objectUrls.add(objectUrl)
      return objectUrl
    })
    resolvedUrlCache.set(url, task)
  }

  return task
}

export function revokeResolvedHeicUrls(): void {
  for (const url of objectUrls) {
    URL.revokeObjectURL(url)
  }
  objectUrls.clear()
  resolvedUrlCache.clear()
}
