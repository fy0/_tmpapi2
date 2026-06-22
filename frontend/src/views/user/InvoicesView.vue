<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="grid gap-3 md:grid-cols-3">
        <div class="card p-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.availableAmount') }}</p>
              <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(summary.available_amount) }}</p>
            </div>
            <Icon name="document" size="lg" class="text-primary-500" />
          </div>
        </div>
        <div class="card p-4">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.pendingAmount') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(summary.pending_amount) }}</p>
        </div>
        <div class="card p-4">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.issuedAmount') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(summary.issued_amount) }}</p>
        </div>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
        <div class="card p-4">
          <div class="mb-4 flex items-center justify-between gap-3">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('invoice.availableRecharges') }}</h2>
            <button class="btn btn-secondary" :disabled="loadingRecharges" :title="t('common.refresh')" @click="loadRecharges">
              <Icon name="refresh" size="md" :class="loadingRecharges ? 'animate-spin' : ''" />
            </button>
          </div>

          <div v-if="loadingRecharges" class="space-y-3">
            <div v-for="i in 4" :key="i" class="h-16 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
          </div>
          <div v-else-if="recharges.length === 0" class="rounded-lg border border-dashed border-gray-200 p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('invoice.noRecharges') }}
          </div>
          <div v-else class="max-h-[420px] space-y-2 overflow-y-auto pr-1">
            <label
              v-for="item in recharges"
              :key="item.id"
              class="flex cursor-pointer items-start gap-3 rounded-lg border border-gray-200 p-3 transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-dark-700 dark:hover:border-primary-700 dark:hover:bg-primary-900/10"
            >
              <input
                type="checkbox"
                class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="selectedIDs.has(item.id)"
                @change="toggleRecharge(item.id)"
              />
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <span class="break-all font-mono text-sm text-gray-900 dark:text-white">{{ item.code }}</span>
                  <span class="text-sm font-semibold text-emerald-600 dark:text-emerald-400">{{ formatMoney(item.value) }}</span>
                </div>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(item.used_at || item.created_at) }}</p>
              </div>
            </label>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('invoice.createRequest') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="input-label">{{ t('invoice.invoiceTitle') }}</label>
              <input v-model.trim="form.invoice_title" class="input mt-1" maxlength="200" />
            </div>
            <div>
              <label class="input-label">{{ t('invoice.taxNo') }}</label>
              <input v-model.trim="form.tax_no" class="input mt-1" maxlength="100" />
            </div>
            <div>
              <label class="input-label">{{ t('invoice.note') }}</label>
              <textarea v-model.trim="form.note" rows="3" class="input mt-1 resize-none" maxlength="1000"></textarea>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
              <div class="flex items-center justify-between text-sm">
                <span class="text-gray-500 dark:text-dark-400">{{ t('invoice.selectedCount') }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ selectedIDs.size }}</span>
              </div>
              <div class="mt-2 flex items-center justify-between text-sm">
                <span class="text-gray-500 dark:text-dark-400">{{ t('invoice.selectedAmount') }}</span>
                <span class="font-semibold text-gray-900 dark:text-white">{{ formatMoney(selectedAmount) }}</span>
              </div>
            </div>
            <button class="btn btn-primary w-full" :disabled="submitting || !canSubmit" @click="submitInvoice">
              <Icon name="plus" size="sm" />
              <span>{{ submitting ? t('common.processing') : t('invoice.submitRequest') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div class="card p-4">
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('invoice.history') }}</h2>
          <div class="flex flex-1 items-center justify-end gap-2">
            <Select v-model="filters.status" :options="statusOptions" class="w-36" @change="loadInvoices" />
            <button class="btn btn-secondary" :disabled="loadingInvoices" :title="t('common.refresh')" @click="loadInvoices">
              <Icon name="refresh" size="md" :class="loadingInvoices ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full min-w-[760px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.id') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.invoiceTitle') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.amount') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.status') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('invoice.createdAt') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loadingInvoices">
                <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else-if="invoices.length === 0">
                <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.noInvoices') }}</td>
              </tr>
              <template v-else>
                <tr v-for="invoice in invoices" :key="invoice.id" class="hover:bg-gray-50 dark:hover:bg-dark-800">
                  <td class="whitespace-nowrap px-4 py-3 font-mono text-sm text-gray-700 dark:text-gray-300">#{{ invoice.id }}</td>
                  <td class="px-4 py-3">
                    <div class="max-w-xs truncate text-sm font-medium text-gray-900 dark:text-white">{{ invoice.invoice_title }}</div>
                    <div v-if="invoice.tax_no" class="max-w-xs truncate text-xs text-gray-500 dark:text-dark-400">{{ invoice.tax_no }}</div>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-sm font-semibold text-gray-900 dark:text-white">{{ formatMoney(invoice.amount) }}</td>
                  <td class="whitespace-nowrap px-4 py-3"><span :class="statusBadgeClass(invoice.status)">{{ statusLabel(invoice.status) }}</span></td>
                  <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(invoice.created_at) }}</td>
                  <td class="whitespace-nowrap px-4 py-3">
                    <button v-if="invoice.status === 'issued' && invoice.file_name" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20" @click="downloadInvoice(invoice)">
                      <Icon name="download" size="sm" />
                      <span>{{ t('invoice.download') }}</span>
                    </button>
                    <span v-else class="text-xs text-gray-400 dark:text-dark-500">-</span>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import invoicesAPI, { type InvoiceRecharge, type InvoiceRequest, type InvoiceStatus } from '@/api/invoices'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const summary = reactive({ available_amount: 0, pending_amount: 0, issued_amount: 0 })
const recharges = ref<InvoiceRecharge[]>([])
const invoices = ref<InvoiceRequest[]>([])
const selectedIDs = ref<Set<number>>(new Set())
const loadingRecharges = ref(false)
const loadingInvoices = ref(false)
const submitting = ref(false)
const filters = reactive({ status: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const form = reactive({ invoice_title: '', tax_no: '', note: '' })

const statusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'pending', label: t('invoice.statuses.pending') },
  { value: 'issued', label: t('invoice.statuses.issued') },
  { value: 'rejected', label: t('invoice.statuses.rejected') },
])

const selectedAmount = computed(() => {
  return recharges.value.reduce((sum, item) => selectedIDs.value.has(item.id) ? sum + item.value : sum, 0)
})

const canSubmit = computed(() => form.invoice_title.trim() !== '' && selectedIDs.value.size > 0 && selectedAmount.value > 0)

function formatMoney(amount: number): string {
  return formatCurrency(amount || 0, 'USD')
}

function toggleRecharge(id: number) {
  const next = new Set(selectedIDs.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIDs.value = next
}

async function loadSummary() {
  try {
    const res = await invoicesAPI.getSummary()
    Object.assign(summary, res.data)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  }
}

async function loadRecharges() {
  loadingRecharges.value = true
  try {
    const res = await invoicesAPI.getRecharges()
    recharges.value = res.data.items || []
    const validIDs = new Set(recharges.value.map(item => item.id))
    selectedIDs.value = new Set([...selectedIDs.value].filter(id => validIDs.has(id)))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    loadingRecharges.value = false
  }
}

async function loadInvoices() {
  loadingInvoices.value = true
  try {
    const res = await invoicesAPI.list({
      page: pagination.page,
      page_size: pagination.page_size,
      status: filters.status || undefined,
    })
    invoices.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    loadingInvoices.value = false
  }
}

async function submitInvoice() {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    await invoicesAPI.create({
      invoice_title: form.invoice_title.trim(),
      tax_no: form.tax_no.trim() || undefined,
      note: form.note.trim() || undefined,
      redeem_code_ids: [...selectedIDs.value],
    })
    appStore.showSuccess(t('invoice.createSuccess'))
    form.invoice_title = ''
    form.tax_no = ''
    form.note = ''
    selectedIDs.value = new Set()
    pagination.page = 1
    await Promise.all([loadSummary(), loadRecharges(), loadInvoices()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('common.error')))
  } finally {
    submitting.value = false
  }
}

async function downloadInvoice(invoice: InvoiceRequest) {
  try {
    const res = await invoicesAPI.download(invoice.id)
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

onMounted(async () => {
  await Promise.all([loadSummary(), loadRecharges(), loadInvoices()])
})
</script>
