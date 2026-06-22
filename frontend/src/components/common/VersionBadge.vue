<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-lg bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-400"
    :title="versionTitle || undefined"
  >
    <span v-if="currentVersion" class="font-medium">v{{ currentVersion }}</span>
    <span v-else class="h-3 w-10 animate-pulse rounded bg-gray-200 dark:bg-dark-600"></span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores'

const props = defineProps<{
  version?: string
}>()

const appStore = useAppStore()

const currentVersion = computed(() => appStore.currentVersion || props.version || '')
const versionTitle = computed(() => (currentVersion.value ? `v${currentVersion.value}` : ''))
</script>
