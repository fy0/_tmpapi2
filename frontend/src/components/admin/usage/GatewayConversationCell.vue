<template>
  <div class="min-w-[180px] max-w-xl space-y-1 text-xs" @click.stop>
    <div v-if="row.conversation_id" class="flex items-center gap-1.5">
      <RouterLink v-if="scopedLink" :to="scopedLink" class="truncate font-mono text-primary-600 hover:underline dark:text-primary-400" :title="row.conversation_id" data-testid="conversation-filter-link">{{ row.conversation_id }}</RouterLink>
      <span v-else class="truncate font-mono" :title="row.conversation_id">{{ row.conversation_id }}</span>
      <button type="button" class="shrink-0 rounded p-1 hover:bg-gray-100 dark:hover:bg-dark-700" :title="t('keys.copyToClipboard')" :aria-label="t('keys.copyToClipboard')" data-testid="conversation-copy" @click="copyConversation"><Icon :name="copied ? 'check' : 'copy'" size="sm" /></button>
    </div>
    <span v-else class="text-gray-400" data-testid="conversation-unknown">—</span>
    <div v-if="row.conversation_id_source" class="break-all text-gray-500 dark:text-gray-400">
      <span v-if="row.conversation_id_source.startsWith('fallback.')" class="text-amber-600">{{ t('admin.usage.correlationFallback') }} · </span>{{ row.conversation_id_source }}
    </div>
    <div v-if="row.attempt_count" class="text-gray-500 dark:text-gray-400">{{ t('admin.usage.recoveryAttempts', { count: row.attempt_count }) }}<span v-if="row.recovery_outcome"> · {{ row.recovery_outcome }}</span></div>
    <details v-if="row.request_started_at || row.attempts?.length" class="text-gray-600 dark:text-gray-300">
      <summary class="cursor-pointer">{{ t('admin.usage.recoveryDetails') }}</summary>
      <dl class="mt-2 space-y-1 break-all">
        <div>{{ t('admin.usage.requestId') }}: <span class="font-mono">{{ row.downstream_request_id || '—' }}</span></div>
        <div>{{ t('admin.usage.requestBodyBytes') }}: {{ row.request_body_bytes ?? '—' }}</div>
        <div>{{ t('admin.usage.requestStartedAt') }}: {{ row.request_started_at || '—' }}</div>
        <div>{{ t('admin.usage.requestEndedAt') }}: {{ row.request_ended_at || '—' }}</div>
        <div>{{ t('admin.usage.downstreamStatus') }}: {{ row.downstream_status_code || '—' }}</div>
      </dl>
      <ol v-if="row.attempts?.length" class="mt-2 space-y-2">
        <li v-for="attempt in row.attempts" :key="attempt.attempt" class="rounded border border-gray-200 p-2 dark:border-dark-600">
          <div>#{{ attempt.attempt }} · {{ attempt.outcome }}</div>
          <div>{{ t('admin.usage.account') }}: {{ attempt.account_id }} · {{ attempt.model || '—' }}</div>
          <div>{{ t('admin.usage.upstreamStatus') }}: {{ attempt.upstream_status || '—' }} · {{ attempt.failure_kind || '—' }}</div>
          <div>{{ t('admin.usage.recoveryRule') }}: {{ attempt.rule_id || '—' }} · {{ attempt.action || '—' }}</div>
          <div class="break-all">{{ attempt.started_at }} → {{ attempt.ended_at }}</div>
          <div>{{ t('admin.usage.attemptCost') }}: {{ attempt.cost_accounting === 'unknown' ? t('admin.usage.attemptCostUnknown') : (attempt.cost_accounting || '—') }}</div>
        </li>
      </ol>
      <p class="mt-2 text-gray-500">{{ t('admin.usage.correlationNotice') }}</p>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'
import type { GatewayRequestMetadata } from '@/types'

const props = defineProps<{ row: GatewayRequestMetadata & { user_id?: number | null; api_key_id?: number | null } }>()
const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const copied = ref(false)
const scopedLink = computed(() => {
  const { conversation_id: id, conversation_id_source: source, user_id: userId, api_key_id: keyId } = props.row
  // A client-controlled ID alone is not an identity or an authorization scope.
  if (!id || !userId || !keyId) return null
  return { path: '/admin/usage', query: { ...route.query, conversation_id: id, conversation_id_source: source || undefined, user_id: String(userId), api_key_id: String(keyId) } }
})
watch(() => props.row.conversation_id, () => { copied.value = false })
async function copyConversation() {
  if (!props.row.conversation_id) return
  try {
    await navigator.clipboard.writeText(props.row.conversation_id)
    copied.value = true
    appStore.showSuccess(t('admin.usage.conversationIdCopied'))
  } catch { appStore.showError(t('common.copyFailed')) }
}
</script>
