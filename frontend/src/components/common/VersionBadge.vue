<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-lg bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-400"
    :title="versionTitle || undefined"
  >
    <span class="font-medium">{{ displayVersion }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores'

const props = defineProps<{
  version?: string
}>()

const appStore = useAppStore()

const pinnedVersion = 'v0.1.138'
const currentVersion = computed(() => appStore.currentVersion || props.version || pinnedVersion)
const displayVersion = computed(() => {
  const value = currentVersion.value.trim()
  if (!value) return pinnedVersion
  return value.startsWith('v') ? value : `v${value}`
})
const versionTitle = computed(() => displayVersion.value)
</script>
