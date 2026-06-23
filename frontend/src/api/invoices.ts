import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'

export type InvoiceStatus = 'pending' | 'issued' | 'rejected'

export interface InvoiceSummary {
  available_amount: number
  pending_amount: number
  issued_amount: number
  min_invoice_amount: number
}

export interface InvoiceRecharge {
  id: number
  code: string
  type: string
  value: number
  used_at?: string | null
  created_at: string
}

export interface InvoiceRequest {
  id: number
  user_id: number
  user_email?: string
  status: InvoiceStatus
  invoice_title: string
  tax_no: string
  amount: number
  note: string
  admin_note: string
  file_name?: string
  content_type?: string
  file_size?: number
  uploaded_by?: number | null
  issued_at?: string | null
  rejected_at?: string | null
  created_at: string
  updated_at: string
  recharges?: InvoiceRecharge[]
}

export interface CreateInvoiceRequest {
  invoice_title: string
  tax_no?: string
  note?: string
  redeem_code_ids: number[]
}

export interface InvoiceListParams {
  page?: number
  page_size?: number
  status?: string
  keyword?: string
}

export const invoicesAPI = {
  getSummary() {
    return apiClient.get<InvoiceSummary>('/invoices/summary')
  },

  getRecharges() {
    return apiClient.get<{ items: InvoiceRecharge[] }>('/invoices/recharges')
  },

  list(params?: InvoiceListParams) {
    return apiClient.get<BasePaginationResponse<InvoiceRequest>>('/invoices', { params })
  },

  create(data: CreateInvoiceRequest) {
    return apiClient.post<InvoiceRequest>('/invoices', data)
  },

  download(id: number) {
    return apiClient.get<Blob>(`/invoices/${id}/download`, { responseType: 'blob' })
  }
}

export default invoicesAPI
