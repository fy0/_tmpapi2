import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'
import type { InvoiceListParams, InvoiceRequest } from '@/api/invoices'

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
  }
}

export default adminInvoicesAPI
