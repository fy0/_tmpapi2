<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-end gap-3">
          <div class="flex-1 sm:max-w-xs">
            <label class="input-label">{{ t('invoice.minInvoiceAmount') }}</label>
            <input
              :value="settingsForm.min_invoice_amount || ''"
              type="number"
              min="0"
              step="0.01"
              class="input mt-1"
              :placeholder="formatMoney(0)"
              @input="settingsForm.min_invoice_amount = Number(($event.target as HTMLInputElement).value) || 0"
            />
          </div>
          <div class="flex-1 sm:max-w-xs">
            <label class="input-label">{{ t('invoice.maxInvoiceAmount') }}</label>
            <input
              :value="settingsForm.max_invoice_amount || ''"
              type="number"
              min="0"
              step="0.01"
              class="input mt-1"
              :placeholder="formatMoney(0)"
              @input="settingsForm.max_invoice_amount = Number(($event.target as HTMLInputElement).value) || 0"
            />
          </div>
          <button class="btn btn-primary" :disabled="savingSettings" @click="saveSettings">
            <Icon name="check" size="sm" />
            <span>{{ savingSettings ? t('common.processing') : t('invoice.saveSettings') }}</span>
          </button>
          <p class="basis-full text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.invoiceAmountLimitHint') }}</p>
        </div>
      </div>

      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-72">
            <input v-model.trim="filters.keyword" class="input" :placeholder="t('invoice.searchPlaceholder')" @keyup.enter="reloadInvoices" />
          </div>
          <Select v-model="filters.status" :options="statusOptions" class="w-36" @change="reloadInvoices" />
          <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300">
            <input v-model="filters.exportOlderThanSixHours" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('invoice.exportOlderThanSixHours') }}</span>
          </label>
          <button class="btn btn-secondary" :disabled="exporting" @click="exportPendingInvoices">
            <Icon name="download" size="sm" />
            <span>{{ exporting ? t('common.processing') : t('invoice.exportPendingCsv') }}</span>
          </button>
          <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadInvoices">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <div class="card p-4">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1180px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.id') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.user') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.invoiceTitle') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.amount') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.status') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.recharges') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.createdAt') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loading">
                <td colspan="8" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else-if="invoices.length === 0">
                <td colspan="8" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.noInvoices') }}</td>
              </tr>
              <template v-else>
                <tr v-for="invoice in invoices" :key="invoice.id" class="hover:bg-gray-50 dark:hover:bg-dark-800">
                  <td class="whitespace-nowrap px-4 py-3 font-mono text-sm text-gray-700 dark:text-gray-300">#{{ invoice.id }}</td>
                  <td class="px-4 py-3">
                    <div class="max-w-[180px] truncate text-sm font-medium text-gray-900 dark:text-white">{{ invoice.user_email || `#${invoice.user_id}` }}</div>
                    <div class="text-xs text-gray-500 dark:text-dark-400">#{{ invoice.user_id }}</div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="max-w-[220px] truncate text-sm font-medium text-gray-900 dark:text-white">{{ invoice.invoice_title }}</div>
                    <div v-if="invoice.tax_no" class="max-w-[220px] truncate text-xs text-gray-500 dark:text-dark-400">{{ invoice.tax_no }}</div>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-sm font-semibold text-gray-900 dark:text-white">{{ formatMoney(invoice.amount) }}</td>
                  <td class="whitespace-nowrap px-4 py-3"><span :class="statusBadgeClass(invoice.status)">{{ statusLabel(invoice.status) }}</span></td>
                  <td class="px-4 py-3">
                    <button class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openDetail(invoice)">
                      {{ invoice.recharges?.length || 0 }}
                    </button>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(invoice.created_at) }}</td>
                  <td class="whitespace-nowrap px-4 py-3">
                    <div class="flex items-center gap-1">
                      <button class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700" @click="openDetail(invoice)">
                        <Icon name="eye" size="sm" />
                        <span>{{ t('common.view') }}</span>
                      </button>
                      <button v-if="canUploadInvoice(invoice)" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20" @click="openUpload(invoice)">
                        <Icon name="upload" size="sm" />
                        <span>{{ t('invoice.uploadInvoice') }}</span>
                      </button>
                      <button v-if="invoice.status === 'issued' && invoice.file_name" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20" @click="downloadInvoice(invoice)">
                        <Icon name="download" size="sm" />
                        <span>{{ t('invoice.download') }}</span>
                      </button>
                      <button v-if="invoice.status === 'issued' && invoice.file_name" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-sky-600 hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20" @click="openPreview(invoice)">
                        <Icon name="document" size="sm" />
                        <span>{{ t('invoice.viewFile') }}</span>
                      </button>
                      <button v-if="invoice.file_name" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20" @click="confirmDeleteFile(invoice)">
                        <Icon name="trash" size="sm" />
                        <span>{{ t('invoice.deleteFile') }}</span>
                      </button>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="pagination.total > 0"
          class="mt-4"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </div>
    </div>

    <input ref="fileInput" type="file" class="hidden" @change="handleFileSelected" />

    <BaseDialog :show="!!selectedInvoice" :title="t('invoice.detail')" width="wide" @close="selectedInvoice = null">
      <div v-if="selectedInvoice" class="space-y-4">
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.user') }}</p>
            <p class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ selectedInvoice.user_email || `#${selectedInvoice.user_id}` }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.status') }}</p>
            <p class="mt-1"><span :class="statusBadgeClass(selectedInvoice.status)">{{ statusLabel(selectedInvoice.status) }}</span></p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.invoiceTitle') }}</p>
            <p class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">{{ selectedInvoice.invoice_title }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.taxNo') }}</p>
            <p class="mt-1 break-all text-sm text-gray-900 dark:text-white">{{ selectedInvoice.tax_no || '-' }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.amount') }}</p>
            <p class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ formatMoney(selectedInvoice.amount) }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.createdAt') }}</p>
            <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatDateTime(selectedInvoice.created_at) }}</p>
          </div>
          <div v-if="selectedInvoice.file_name">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.fileName') }}</p>
            <p class="mt-1 break-all text-sm text-gray-900 dark:text-white">{{ selectedInvoice.file_name }}</p>
          </div>
          <div v-if="selectedInvoice.file_size">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.fileSize') }}</p>
            <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatFileSize(selectedInvoice.file_size) }}</p>
          </div>
          <div v-if="selectedInvoice.note" class="md:col-span-2">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('invoice.note') }}</p>
            <p class="mt-1 whitespace-pre-wrap text-sm text-gray-900 dark:text-white">{{ selectedInvoice.note }}</p>
          </div>
        </div>

        <div>
          <p class="mb-2 text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('invoice.selectedRecharges') }}</p>
          <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
            <table class="w-full min-w-[520px] divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.code') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.amount') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.usedAt') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
                <tr v-for="recharge in selectedInvoice.recharges || []" :key="recharge.id">
                  <td class="break-all px-3 py-2 font-mono text-xs text-gray-700 dark:text-gray-300">{{ recharge.code }}</td>
                  <td class="whitespace-nowrap px-3 py-2 text-sm font-medium text-gray-900 dark:text-white">{{ formatMoney(recharge.value) }}</td>
                  <td class="whitespace-nowrap px-3 py-2 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(recharge.used_at || recharge.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="selectedInvoice = null">{{ t('common.close') }}</button>
          <button v-if="selectedInvoice && canUploadInvoice(selectedInvoice)" class="btn btn-primary" @click="openUpload(selectedInvoice)">
            <Icon name="upload" size="sm" />
            <span>{{ t('invoice.uploadInvoice') }}</span>
          </button>
          <button v-if="selectedInvoice?.status === 'issued' && selectedInvoice.file_name" class="btn btn-secondary" @click="openPreview(selectedInvoice)">
            <Icon name="document" size="sm" />
            <span>{{ t('invoice.viewFile') }}</span>
          </button>
          <button v-if="selectedInvoice?.file_name" class="btn btn-secondary text-red-600 dark:text-red-400" @click="confirmDeleteFile(selectedInvoice)">
            <Icon name="trash" size="sm" />
            <span>{{ t('invoice.deleteFile') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!previewInvoice" :title="t('invoice.preview')" width="wide" @close="closePreview">
      <div v-if="previewInvoice" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
          <div class="min-w-0">
            <p class="break-all text-sm font-medium text-gray-900 dark:text-white">{{ previewInvoice.file_name }}</p>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ formatFileSize(previewInvoice.file_size) }}</p>
          </div>
          <button class="btn btn-secondary btn-sm" @click="downloadInvoice(previewInvoice)">
            <Icon name="download" size="sm" />
            <span>{{ t('invoice.download') }}</span>
          </button>
        </div>
        <div class="min-h-[520px] overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
          <div v-if="previewLoading" class="flex h-[520px] items-center justify-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('common.loading') }}
          </div>
          <img v-else-if="previewURL && previewKind === 'image'" :src="previewURL" class="mx-auto max-h-[720px] w-auto max-w-full object-contain" />
          <iframe v-else-if="previewURL && previewKind === 'pdf'" :src="previewURL" class="h-[720px] w-full border-0"></iframe>
          <div v-else class="flex h-[520px] items-center justify-center px-6 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('invoice.previewUnavailable') }}
          </div>
        </div>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="!!pendingUpload"
      :title="t('invoice.uploadConfirmTitle')"
      :message="pendingUpload ? t('invoice.uploadConfirmMessage', { id: pendingUpload.invoice.id, file: pendingUpload.file.name }) : ''"
      :confirm-text="t('invoice.uploadInvoice')"
      @confirm="confirmUpload"
      @cancel="cancelUpload"
    >
      <div v-if="pendingUpload" class="rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
        {{ formatFileSize(pendingUpload.file.size) }}
      </div>
    </ConfirmDialog>

    <ConfirmDialog
      :show="!!deleteTarget"
      :title="t('invoice.deleteFileConfirmTitle')"
      :message="deleteTarget ? t('invoice.deleteFileConfirmMessage', { id: deleteTarget.id }) : ''"
      :confirm-text="t('invoice.deleteFile')"
      :danger="true"
      @confirm="deleteInvoiceFile"
      @cancel="deleteTarget = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import adminInvoicesAPI from '@/api/admin/invoices'
import type { InvoiceRequest, InvoiceStatus } from '@/api/invoices'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const uploading = ref(false)
const exporting = ref(false)
const savingSettings = ref(false)
const deletingFile = ref(false)
const invoices = ref<InvoiceRequest[]>([])
const selectedInvoice = ref<InvoiceRequest | null>(null)
const uploadTarget = ref<InvoiceRequest | null>(null)
const pendingUpload = ref<{ invoice: InvoiceRequest; file: File } | null>(null)
const deleteTarget = ref<InvoiceRequest | null>(null)
const previewInvoice = ref<InvoiceRequest | null>(null)
const previewURL = ref('')
const previewKind = ref<'image' | 'pdf' | 'other'>('other')
const previewLoading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const filters = reactive({ status: '', keyword: '', exportOlderThanSixHours: false })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const settingsForm = reactive({ min_invoice_amount: 0, max_invoice_amount: 0 })

const statusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'pending', label: t('invoice.statuses.pending') },
  { value: 'issued', label: t('invoice.statuses.issued') },
  { value: 'rejected', label: t('invoice.statuses.rejected') },
  { value: 'withdrawn', label: t('invoice.statuses.withdrawn') },
])

function formatMoney(amount: number): string {
  return (Number(amount) || 0).toFixed(2)
}

function formatFileSize(size?: number): string {
  return formatBytes(size || 0)
}

function canUploadInvoice(invoice: InvoiceRequest): boolean {
  return invoice.status === 'pending' || invoice.status === 'issued'
}

async function loadInvoices() {
  loading.value = true
  try {
    const res = await adminInvoicesAPI.list({
      page: pagination.page,
      page_size: pagination.page_size,
      status: filters.status || undefined,
      keyword: filters.keyword || undefined,
    })
    invoices.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function reloadInvoices() {
  pagination.page = 1
  loadInvoices()
}

async function loadSettings() {
  try {
    const res = await adminInvoicesAPI.getSettings()
    settingsForm.min_invoice_amount = res.data.min_invoice_amount || 0
    settingsForm.max_invoice_amount = res.data.max_invoice_amount || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    const res = await adminInvoicesAPI.updateSettings({
      min_invoice_amount: Math.max(0, Number(settingsForm.min_invoice_amount) || 0),
      max_invoice_amount: Math.max(0, Number(settingsForm.max_invoice_amount) || 0),
    })
    settingsForm.min_invoice_amount = res.data.min_invoice_amount || 0
    settingsForm.max_invoice_amount = res.data.max_invoice_amount || 0
    appStore.showSuccess(t('invoice.settingsSaved'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    savingSettings.value = false
  }
}

async function openDetail(invoice: InvoiceRequest) {
  try {
    const res = await adminInvoicesAPI.get(invoice.id)
    selectedInvoice.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  }
}

function openUpload(invoice: InvoiceRequest) {
  uploadTarget.value = invoice
  if (fileInput.value) {
    fileInput.value.value = ''
    fileInput.value.click()
  }
}

async function handleFileSelected(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file || !uploadTarget.value || uploading.value) return
  pendingUpload.value = { invoice: uploadTarget.value, file }
  uploadTarget.value = null
  target.value = ''
}

async function confirmUpload() {
  if (!pendingUpload.value || uploading.value) return
  const { invoice, file } = pendingUpload.value
  uploading.value = true
  try {
    const res = await adminInvoicesAPI.upload(invoice.id, file)
    appStore.showSuccess(t('invoice.uploadSuccess'))
    selectedInvoice.value = selectedInvoice.value?.id === res.data.id ? res.data : selectedInvoice.value
    if (previewInvoice.value?.id === res.data.id) closePreview()
    pendingUpload.value = null
    await loadInvoices()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    uploading.value = false
  }
}

function cancelUpload() {
  pendingUpload.value = null
}

async function downloadInvoice(invoice: InvoiceRequest) {
  try {
    const res = await adminInvoicesAPI.download(invoice.id)
    saveBlob(res.data, invoice.file_name || `invoice-${invoice.id}`)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  }
}

async function exportPendingInvoices() {
  exporting.value = true
  try {
    const res = await adminInvoicesAPI.exportPending({
      keyword: filters.keyword || undefined,
      min_age_hours: filters.exportOlderThanSixHours ? 6 : undefined,
    })
    saveBlob(res.data, `pending-invoices-${new Date().toISOString().slice(0, 10)}.csv`)
    appStore.showSuccess(t('invoice.exportSuccess'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    exporting.value = false
  }
}

async function openPreview(invoice: InvoiceRequest) {
  closePreview()
  previewInvoice.value = invoice
  previewLoading.value = true
  previewKind.value = previewKindFromInvoice(invoice)
  try {
    const res = await adminInvoicesAPI.preview(invoice.id)
    previewURL.value = URL.createObjectURL(res.data)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
    previewInvoice.value = null
  } finally {
    previewLoading.value = false
  }
}

function closePreview() {
  if (previewURL.value) {
    URL.revokeObjectURL(previewURL.value)
  }
  previewURL.value = ''
  previewInvoice.value = null
  previewKind.value = 'other'
  previewLoading.value = false
}

function previewKindFromInvoice(invoice: InvoiceRequest): 'image' | 'pdf' | 'other' {
  const contentType = (invoice.content_type || '').toLowerCase()
  const fileName = (invoice.file_name || '').toLowerCase()
  if (contentType.startsWith('image/') || /\.(png|jpe?g|gif|webp|bmp)$/i.test(fileName)) return 'image'
  if (contentType.includes('pdf') || fileName.endsWith('.pdf')) return 'pdf'
  return 'other'
}

function confirmDeleteFile(invoice: InvoiceRequest) {
  deleteTarget.value = invoice
}

async function deleteInvoiceFile() {
  if (!deleteTarget.value || deletingFile.value) return
  deletingFile.value = true
  try {
    const res = await adminInvoicesAPI.deleteFile(deleteTarget.value.id)
    appStore.showSuccess(t('invoice.deleteFileSuccess'))
    selectedInvoice.value = selectedInvoice.value?.id === res.data.id ? res.data : selectedInvoice.value
    if (previewInvoice.value?.id === res.data.id) closePreview()
    deleteTarget.value = null
    await loadInvoices()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    deletingFile.value = false
  }
}

function saveBlob(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function handlePageChange(page: number) {
  pagination.page = page
  loadInvoices()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  loadInvoices()
}

function statusLabel(status: InvoiceStatus): string {
  return t(`invoice.statuses.${status}`)
}

function statusBadgeClass(status: InvoiceStatus): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  if (status === 'issued') return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
  if (status === 'rejected') return `${base} bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300`
  if (status === 'withdrawn') return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300`
  return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
}

onMounted(async () => {
  await Promise.all([loadSettings(), loadInvoices()])
})

onBeforeUnmount(() => closePreview())
</script>
