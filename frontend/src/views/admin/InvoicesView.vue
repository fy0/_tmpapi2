<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-72">
            <input v-model.trim="filters.keyword" class="input" :placeholder="t('invoice.searchPlaceholder')" @keyup.enter="reloadInvoices" />
          </div>
          <Select v-model="filters.status" :options="statusOptions" class="w-36" @change="reloadInvoices" />
          <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadInvoices">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <div class="card p-4">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1040px] divide-y divide-gray-200 dark:divide-dark-700">
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
                      <button class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20" @click="openUpload(invoice)">
                        <Icon name="upload" size="sm" />
                        <span>{{ t('invoice.uploadInvoice') }}</span>
                      </button>
                      <button v-if="invoice.status === 'issued' && invoice.file_name" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20" @click="downloadInvoice(invoice)">
                        <Icon name="download" size="sm" />
                        <span>{{ t('invoice.download') }}</span>
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
          <button v-if="selectedInvoice" class="btn btn-primary" @click="openUpload(selectedInvoice)">
            <Icon name="upload" size="sm" />
            <span>{{ t('invoice.uploadInvoice') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import adminInvoicesAPI from '@/api/admin/invoices'
import type { InvoiceRequest, InvoiceStatus } from '@/api/invoices'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatBytes, formatCurrency, formatDateTime } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const uploading = ref(false)
const invoices = ref<InvoiceRequest[]>([])
const selectedInvoice = ref<InvoiceRequest | null>(null)
const uploadTarget = ref<InvoiceRequest | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const filters = reactive({ status: '', keyword: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'pending', label: t('invoice.statuses.pending') },
  { value: 'issued', label: t('invoice.statuses.issued') },
  { value: 'rejected', label: t('invoice.statuses.rejected') },
])

function formatMoney(amount: number): string {
  return formatCurrency(amount || 0, 'USD')
}

function formatFileSize(size?: number): string {
  return formatBytes(size || 0)
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
  uploading.value = true
  try {
    const res = await adminInvoicesAPI.upload(uploadTarget.value.id, file)
    appStore.showSuccess(t('invoice.uploadSuccess'))
    selectedInvoice.value = selectedInvoice.value?.id === res.data.id ? res.data : selectedInvoice.value
    await loadInvoices()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    uploading.value = false
    uploadTarget.value = null
    target.value = ''
  }
}

async function downloadInvoice(invoice: InvoiceRequest) {
  try {
    const res = await adminInvoicesAPI.download(invoice.id)
    saveBlob(res.data, invoice.file_name || `invoice-${invoice.id}`)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
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
  return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
}

onMounted(() => loadInvoices())
</script>
