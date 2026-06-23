import { apiClient } from './client'
import type {
  AddSupportTicketMessageRequest,
  BasePaginationResponse,
  CreateSupportTicketRequest,
  SupportTicket
} from '@/types'

export interface SupportTicketListFilters {
  status?: string
  category?: string
  search?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: SupportTicketListFilters,
  options?: { signal?: AbortSignal }
): Promise<BasePaginationResponse<SupportTicket>> {
  const { data } = await apiClient.get<BasePaginationResponse<SupportTicket>>('/support-tickets', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<SupportTicket> {
  const { data } = await apiClient.get<SupportTicket>(`/support-tickets/${id}`)
  return data
}

export async function create(request: CreateSupportTicketRequest): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>('/support-tickets', request)
  return data
}

export async function addMessage(
  id: number,
  request: AddSupportTicketMessageRequest
): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>(`/support-tickets/${id}/messages`, request)
  return data
}

const supportTicketsAPI = {
  list,
  getById,
  create,
  addMessage
}

export default supportTicketsAPI
