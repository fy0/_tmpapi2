import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'
import type { InvoiceListParams, InvoiceRequest } from '@/api/invoices'

export interface InvoiceSettings {
  min_invoice_amount: number
}

export const adminInvoicesAPI = {
  list(params?: InvoiceListParams) {
    return apiClient.get<BasePaginationResponse<InvoiceRequest>>('/admin/invoices', { params })
  },

  get(id: number) {
    return apiClient.get<InvoiceRequest>(`/admin/invoices/${id}`)
  },

  upload(id: number, file: File) {
    const formData = new FormData()
    formData.append('file', file)
    return apiClient.post<InvoiceRequest>(`/admin/invoices/${id}/upload`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  download(id: number) {
    return apiClient.get<Blob>(`/admin/invoices/${id}/download`, { responseType: 'blob' })
  },

  preview(id: number) {
    return apiClient.get<Blob>(`/admin/invoices/${id}/download`, {
      params: { inline: 1 },
      responseType: 'blob'
    })
  },

  deleteFile(id: number) {
    return apiClient.delete<InvoiceRequest>(`/admin/invoices/${id}/file`)
  },

  exportPending(params?: Pick<InvoiceListParams, 'keyword'>) {
    return apiClient.get<Blob>('/admin/invoices/export', {
      params,
      responseType: 'blob'
    })
  },

  getSettings() {
    return apiClient.get<InvoiceSettings>('/admin/invoices/settings')
  },

  updateSettings(data: InvoiceSettings) {
    return apiClient.put<InvoiceSettings>('/admin/invoices/settings', data)
  }
}

export default adminInvoicesAPI
