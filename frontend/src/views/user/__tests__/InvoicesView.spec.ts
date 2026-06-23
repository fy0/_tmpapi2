import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import InvoicesView from '../InvoicesView.vue'

const { getSummary, getProfile, getRecharges, list, create, download, showError, showSuccess } = vi.hoisted(() => ({
  getSummary: vi.fn(),
  getProfile: vi.fn(),
  getRecharges: vi.fn(),
  list: vi.fn(),
  create: vi.fn(),
  download: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/invoices', () => ({
  default: {
    getSummary,
    getProfile,
    getRecharges,
    list,
    create,
    download,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const SelectStub = { template: '<select />', props: ['modelValue', 'options'] }
const PaginationStub = { template: '<nav />' }
const IconStub = { template: '<span />', props: ['name', 'size'] }

function mountView() {
  return mount(InvoicesView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Select: SelectStub,
        Pagination: PaginationStub,
        Icon: IconStub,
      },
    },
  })
}

describe('user InvoicesView', () => {
  beforeEach(() => {
    getSummary.mockReset()
    getProfile.mockReset()
    getRecharges.mockReset()
    list.mockReset()
    create.mockReset()
    download.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    getSummary.mockResolvedValue({
      data: {
        available_amount: 0,
        pending_amount: 0,
        issued_amount: 0,
        min_invoice_amount: 0,
      },
    })
    getProfile.mockResolvedValue({
      data: {
        invoice_title: 'ACME Ltd',
        tax_no: 'TAX-123',
        updated_at: '2026-06-23T00:00:00Z',
      },
    })
    getRecharges.mockResolvedValue({ data: { items: [] } })
    list.mockResolvedValue({
      data: {
        items: [{
          id: 99,
          user_id: 7,
          status: 'issued',
          invoice_title: 'ACME Ltd',
          tax_no: 'TAX-123',
          amount: 20,
          note: '',
          admin_note: '',
          file_name: '',
          created_at: '2026-06-23T00:00:00Z',
          updated_at: '2026-06-23T00:00:00Z',
        }],
        total: 1,
      },
    })
  })

  it('prefills saved invoice profile and renders invoice amounts without currency prefix', async () => {
    const wrapper = mountView()
    await flushPromises()

    const inputs = wrapper.findAll('input')
    expect((inputs[0].element as HTMLInputElement).value).toBe('ACME Ltd')
    expect((inputs[1].element as HTMLInputElement).value).toBe('TAX-123')
    expect(wrapper.text()).toContain('20.00')
    expect(wrapper.text()).not.toContain('US$')
  })
})
