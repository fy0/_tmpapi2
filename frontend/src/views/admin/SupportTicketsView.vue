<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 xl:flex-row xl:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-80">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.supportTickets.searchPlaceholder')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>
            <input
              v-model.number="filters.user_id"
              type="number"
              min="1"
              class="input w-32"
              :placeholder="t('admin.supportTickets.userId')"
              @input="handleSearch"
            />
            <Select v-model="filters.status" :options="statusFilterOptions" class="w-40" @change="handleFilterChange" />
            <Select v-model="filters.category" :options="categoryFilterOptions" class="w-40" @change="handleFilterChange" />
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 xl:w-auto">
            <button
              @click="loadTickets"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="openCreateDialog" class="btn btn-primary">
              <Icon name="plus" size="md" />
              <span>{{ t('admin.supportTickets.create') }}</span>
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="tickets"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="last_message_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-title="{ row }">
            <div class="min-w-0">
              <button
                class="max-w-md truncate text-left font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
                @click="openDetail(row)"
              >
                {{ row.title }}
              </button>
              <div class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span>#{{ row.id }}</span>
                <span class="text-gray-300 dark:text-dark-700">·</span>
                <span>{{ formatUnix(row.created_at) }}</span>
              </div>
            </div>
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0 text-sm">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.user_email || row.user_name || `#${row.user_id}` }}</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">#{{ row.user_id }} {{ row.user_name }}</div>
            </div>
          </template>

          <template #cell-category="{ value }">
            <span class="badge badge-gray">{{ categoryLabel(value) }}</span>
          </template>

          <template #cell-status="{ value }">
            <span :class="statusBadgeClass(value)">{{ statusLabel(value) }}</span>
          </template>

          <template #cell-priority="{ value }">
            <span :class="priorityBadgeClass(value)">{{ priorityLabel(value) }}</span>
          </template>

          <template #cell-last_message_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatUnix(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                @click="openDetail(row)"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('admin.supportTickets.view')"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                @click="openEditDialog(row)"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.supportTickets.empty')"
              :description="t('admin.supportTickets.emptyDescription')"
              :action-text="t('admin.supportTickets.create')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog :show="showCreateDialog" :title="t('admin.supportTickets.create')" width="wide" @close="closeCreateDialog">
      <form id="admin-support-ticket-create-form" class="space-y-4" @submit.prevent="createTicket">
        <div>
          <label class="input-label">{{ t('admin.supportTickets.userId') }}</label>
          <input v-model.number="createForm.user_id" type="number" min="1" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('supportTickets.form.title') }}</label>
          <input v-model.trim="createForm.title" type="text" class="input" maxlength="200" required />
        </div>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('supportTickets.form.category') }}</label>
            <Select v-model="createForm.category" :options="categoryOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('supportTickets.form.priority') }}</label>
            <Select v-model="createForm.priority" :options="priorityOptions" />
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('supportTickets.form.content') }}</label>
          <textarea v-model.trim="createForm.content" rows="7" class="input resize-none" maxlength="4000" required></textarea>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeCreateDialog">{{ t('common.cancel') }}</button>
          <button type="submit" form="admin-support-ticket-create-form" class="btn btn-primary" :disabled="saving">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="showEditDialog" :title="t('admin.supportTickets.edit')" width="normal" @close="closeEditDialog">
      <form id="admin-support-ticket-edit-form" class="space-y-4" @submit.prevent="updateTicket">
        <div>
          <label class="input-label">{{ t('supportTickets.form.status') }}</label>
          <Select v-model="editForm.status" :options="statusOptions" />
        </div>
        <div>
          <label class="input-label">{{ t('supportTickets.form.priority') }}</label>
          <Select v-model="editForm.priority" :options="priorityOptions" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeEditDialog">{{ t('common.cancel') }}</button>
          <button type="submit" form="admin-support-ticket-edit-form" class="btn btn-primary" :disabled="saving">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showDetailDialog"
      :title="detailTicket ? `#${detailTicket.id} ${detailTicket.title}` : t('supportTickets.detail')"
      width="extra-wide"
      @close="closeDetail"
    >
      <div v-if="detailTicket" class="space-y-4">
        <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-start">
          <div class="flex flex-wrap items-center gap-2">
            <span :class="statusBadgeClass(detailTicket.status)">{{ statusLabel(detailTicket.status) }}</span>
            <span :class="priorityBadgeClass(detailTicket.priority)">{{ priorityLabel(detailTicket.priority) }}</span>
            <span class="badge badge-gray">{{ categoryLabel(detailTicket.category) }}</span>
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ detailTicket.user_email || detailTicket.user_name || `#${detailTicket.user_id}` }}</span>
          </div>
          <button class="btn btn-secondary btn-sm" @click="openEditDialog(detailTicket)">
            <Icon name="edit" size="sm" />
            <span>{{ t('common.edit') }}</span>
          </button>
        </div>

        <div class="max-h-[52vh] space-y-3 overflow-y-auto pr-1">
          <div
            v-for="message in detailTicket.messages || []"
            :key="message.id"
            class="rounded-lg border p-4"
            :class="message.author_role === 'admin'
              ? 'border-primary-200 bg-primary-50/60 dark:border-primary-800 dark:bg-primary-900/20'
              : 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900'"
          >
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div class="min-w-0">
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ message.author_name || message.author_email || roleLabel(message.author_role) }}
                </span>
                <span class="ml-2 text-xs text-gray-500 dark:text-dark-400">{{ roleLabel(message.author_role) }}</span>
              </div>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ formatUnix(message.created_at) }}</span>
            </div>
            <p class="whitespace-pre-wrap break-words text-sm leading-6 text-gray-700 dark:text-gray-200">{{ message.content }}</p>
          </div>
        </div>

        <form class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700" @submit.prevent="replyTicket">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_220px]">
            <div>
              <label class="input-label">{{ t('supportTickets.reply') }}</label>
              <textarea v-model.trim="replyContent" rows="4" class="input resize-none" maxlength="4000" required></textarea>
            </div>
            <div>
              <label class="input-label">{{ t('supportTickets.form.status') }}</label>
              <Select v-model="replyStatus" :options="statusOptions" />
            </div>
          </div>
          <div class="flex justify-end">
            <button type="submit" class="btn btn-primary" :disabled="replying || !replyContent.trim()">
              {{ replying ? t('common.saving') : t('supportTickets.sendReply') }}
            </button>
          </div>
        </form>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { SupportTicket } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const tickets = ref<SupportTicket[]>([])
const loading = ref(false)
const saving = ref(false)
const replying = ref(false)
const searchQuery = ref('')
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDetailDialog = ref(false)
const detailTicket = ref<SupportTicket | null>(null)
const editingTicket = ref<SupportTicket | null>(null)
const replyContent = ref('')
const replyStatus = ref('pending')

const filters = reactive({
  user_id: null as number | null,
  status: '',
  category: ''
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 0
})

const sortState = reactive({
  sort_by: 'last_message_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const createForm = reactive({
  user_id: null as number | null,
  title: '',
  category: 'feedback',
  priority: 'normal',
  content: ''
})

const editForm = reactive({
  status: 'open',
  priority: 'normal'
})

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('supportTickets.columns.title'), sortable: true },
  { key: 'user', label: t('admin.supportTickets.user') },
  { key: 'category', label: t('supportTickets.columns.category'), sortable: true },
  { key: 'status', label: t('supportTickets.columns.status'), sortable: true },
  { key: 'priority', label: t('supportTickets.columns.priority'), sortable: true },
  { key: 'last_message_at', label: t('supportTickets.columns.lastMessageAt'), sortable: true },
  { key: 'actions', label: t('common.actions') }
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('supportTickets.filters.allStatus') },
  ...statusOptions.value
])

const categoryFilterOptions = computed(() => [
  { value: '', label: t('supportTickets.filters.allCategories') },
  ...categoryOptions.value
])

const statusOptions = computed(() => [
  { value: 'open', label: t('supportTickets.status.open') },
  { value: 'pending', label: t('supportTickets.status.pending') },
  { value: 'resolved', label: t('supportTickets.status.resolved') },
  { value: 'closed', label: t('supportTickets.status.closed') }
])

const categoryOptions = computed(() => [
  { value: 'feedback', label: t('supportTickets.category.feedback') },
  { value: 'bug', label: t('supportTickets.category.bug') },
  { value: 'billing', label: t('supportTickets.category.billing') },
  { value: 'account', label: t('supportTickets.category.account') },
  { value: 'other', label: t('supportTickets.category.other') }
])

const priorityOptions = computed(() => [
  { value: 'low', label: t('supportTickets.priority.low') },
  { value: 'normal', label: t('supportTickets.priority.normal') },
  { value: 'high', label: t('supportTickets.priority.high') }
])

let currentController: AbortController | null = null
let searchDebounceTimer: number | null = null

function formatUnix(value: number | string | null | undefined): string {
  if (!value) return '-'
  const timestamp = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(timestamp) || timestamp <= 0) return '-'
  return formatDateTime(new Date(timestamp * 1000))
}

function statusLabel(status: string): string {
  return t(`supportTickets.status.${status}`, status)
}

function categoryLabel(category: string): string {
  return t(`supportTickets.category.${category}`, category)
}

function priorityLabel(priority: string): string {
  return t(`supportTickets.priority.${priority}`, priority)
}

function roleLabel(role: string): string {
  return t(`supportTickets.role.${role}`, role)
}

function statusBadgeClass(status: string): string {
  if (status === 'resolved') return 'badge badge-success'
  if (status === 'closed') return 'badge badge-gray'
  if (status === 'pending') return 'badge badge-warning'
  return 'badge badge-primary'
}

function priorityBadgeClass(priority: string): string {
  if (priority === 'high') return 'badge badge-danger'
  if (priority === 'low') return 'badge badge-gray'
  return 'badge badge-primary'
}

async function loadTickets() {
  currentController?.abort()
  const requestController = new AbortController()
  currentController = requestController
  const { signal } = requestController
  loading.value = true
  try {
    const res = await adminAPI.supportTickets.list(
      pagination.page,
      pagination.page_size,
      {
        user_id: filters.user_id && filters.user_id > 0 ? filters.user_id : undefined,
        status: filters.status || undefined,
        category: filters.category || undefined,
        search: searchQuery.value.trim() || undefined,
        sort_by: sortState.sort_by,
        sort_order: sortState.sort_order
      },
      { signal }
    )
    if (signal.aborted || currentController !== requestController) return
    tickets.value = res.items
    pagination.total = res.total
    pagination.pages = res.pages
    pagination.page = res.page
    pagination.page_size = res.page_size
  } catch (err: unknown) {
    if ((err as { code?: string; name?: string })?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.supportTickets.failedToLoad')))
  } finally {
    if (currentController === requestController) {
      loading.value = false
      currentController = null
    }
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadTickets()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadTickets()
}

function handleFilterChange() {
  pagination.page = 1
  loadTickets()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadTickets()
}

function handleSearch() {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  searchDebounceTimer = window.setTimeout(() => {
    pagination.page = 1
    loadTickets()
  }, 300)
}

function openCreateDialog() {
  createForm.user_id = null
  createForm.title = ''
  createForm.category = 'feedback'
  createForm.priority = 'normal'
  createForm.content = ''
  showCreateDialog.value = true
}

function closeCreateDialog() {
  showCreateDialog.value = false
}

async function createTicket() {
  if (!createForm.user_id || createForm.user_id <= 0 || !createForm.title.trim() || !createForm.content.trim()) return
  saving.value = true
  try {
    const created = await adminAPI.supportTickets.create({
      user_id: createForm.user_id,
      title: createForm.title.trim(),
      category: createForm.category,
      priority: createForm.priority,
      content: createForm.content.trim()
    })
    appStore.showSuccess(t('admin.supportTickets.created'))
    showCreateDialog.value = false
    pagination.page = 1
    await loadTickets()
    await openDetail(created)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.supportTickets.failedToCreate')))
  } finally {
    saving.value = false
  }
}

function openEditDialog(ticket: SupportTicket) {
  editingTicket.value = ticket
  editForm.status = ticket.status
  editForm.priority = ticket.priority
  showEditDialog.value = true
}

function closeEditDialog() {
  showEditDialog.value = false
  editingTicket.value = null
}

async function updateTicket() {
  if (!editingTicket.value) return
  saving.value = true
  try {
    const updated = await adminAPI.supportTickets.update(editingTicket.value.id, {
      status: editForm.status,
      priority: editForm.priority
    })
    appStore.showSuccess(t('admin.supportTickets.updated'))
    closeEditDialog()
    if (detailTicket.value?.id === updated.id) {
      detailTicket.value = await adminAPI.supportTickets.getById(updated.id)
      replyStatus.value = detailTicket.value.status
    }
    await loadTickets()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.supportTickets.failedToUpdate')))
  } finally {
    saving.value = false
  }
}

async function openDetail(ticket: SupportTicket) {
  try {
    detailTicket.value = await adminAPI.supportTickets.getById(ticket.id)
    replyContent.value = ''
    replyStatus.value = detailTicket.value.status === 'open' ? 'pending' : detailTicket.value.status
    showDetailDialog.value = true
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.supportTickets.failedToLoadDetail')))
  }
}

function closeDetail() {
  showDetailDialog.value = false
  detailTicket.value = null
  replyContent.value = ''
}

async function replyTicket() {
  if (!detailTicket.value || !replyContent.value.trim()) return
  replying.value = true
  try {
    await adminAPI.supportTickets.addMessage(detailTicket.value.id, {
      content: replyContent.value.trim(),
      status: replyStatus.value
    })
    appStore.showSuccess(t('supportTickets.replied'))
    const id = detailTicket.value.id
    replyContent.value = ''
    detailTicket.value = await adminAPI.supportTickets.getById(id)
    replyStatus.value = detailTicket.value.status
    await loadTickets()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('supportTickets.failedToReply')))
  } finally {
    replying.value = false
  }
}

onMounted(loadTickets)

onUnmounted(() => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  currentController?.abort()
})
</script>
