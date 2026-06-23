import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import adminInvoicesAPI from '@/api/admin/invoices'

describe('admin invoices api', () => {
  beforeEach(() => {
    get.mockReset()
    get.mockResolvedValue({ data: {} })
  })

  it('passes the min age filter to pending invoice export', async () => {
    await adminInvoicesAPI.exportPending({
      keyword: 'acme',
      min_age_hours: 6,
    })

    expect(get).toHaveBeenCalledWith('/admin/invoices/export', {
      params: {
        keyword: 'acme',
        min_age_hours: 6,
      },
      responseType: 'blob',
    })
  })
})
