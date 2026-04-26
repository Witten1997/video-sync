<template>
  <img
    v-if="resolvedSrc && !loadFailed"
    :src="resolvedSrc"
    :alt="alt"
    :class="imgClass"
    :style="imgStyle"
    :width="width"
    :height="height"
    :loading="loading"
    :decoding="decoding"
    @error="loadFailed = true"
  />
  <div v-else :class="fallbackClass" :style="fallbackStyle">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { resolveHeicImageUrl } from '@/utils/heic'

const props = withDefaults(defineProps<{
  src?: string
  alt?: string
  imgClass?: string
  imgStyle?: string | Record<string, string>
  fallbackClass?: string
  fallbackStyle?: string | Record<string, string>
  width?: string | number
  height?: string | number
  loading?: 'eager' | 'lazy'
  decoding?: 'sync' | 'async' | 'auto'
}>(), {
  src: '',
  alt: '',
  imgClass: '',
  imgStyle: '',
  fallbackClass: '',
  fallbackStyle: '',
  width: undefined,
  height: undefined,
  loading: 'lazy',
  decoding: 'async'
})

const resolvedSrc = ref('')
const loadFailed = ref(false)
let version = 0

watch(
  () => props.src,
  async (src) => {
    const currentVersion = ++version
    loadFailed.value = false
    resolvedSrc.value = ''

    if (!src) {
      return
    }

    try {
      const nextSrc = await resolveHeicImageUrl(src)
      if (currentVersion === version) {
        resolvedSrc.value = nextSrc
      }
    } catch (error) {
      if (currentVersion === version) {
        resolvedSrc.value = src
      }
      console.error('图片解码失败:', error)
    }
  },
  { immediate: true }
)
</script>
