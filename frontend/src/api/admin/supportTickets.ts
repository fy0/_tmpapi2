import { apiClient } from '../client'
import type {
  AdminAddSupportTicketMessageRequest,
  AdminCreateSupportTicketRequest,
  BasePaginationResponse,
  SupportTicket,
  UpdateSupportTicketRequest
} from '@/types'

export interface AdminSupportTicketListFilters {
  user_id?: number
  status?: string
  category?: string
  search?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: AdminSupportTicketListFilters,
  options?: { signal?: AbortSignal }
): Promise<BasePaginationResponse<SupportTicket>> {
  const { data } = await apiClient.get<BasePaginationResponse<SupportTicket>>('/admin/support-tickets', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<SupportTicket> {
  const { data } = await apiClient.get<SupportTicket>(`/admin/support-tickets/${id}`)
  return data
}

export async function create(request: AdminCreateSupportTicketRequest): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>('/admin/support-tickets', request)
  return data
}

export async function update(id: number, request: UpdateSupportTicketRequest): Promise<SupportTicket> {
  const { data } = await apiClient.put<SupportTicket>(`/admin/support-tickets/${id}`, request)
  return data
}

export async function addMessage(
  id: number,
  request: AdminAddSupportTicketMessageRequest
): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>(`/admin/support-tickets/${id}/messages`, request)
  return data
}

const supportTicketsAPI = {
  list,
  getById,
  create,
  update,
  addMessage
}

export default supportTicketsAPI
