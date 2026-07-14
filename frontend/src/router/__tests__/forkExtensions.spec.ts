import { beforeAll, describe, expect, it, vi } from 'vitest'

const routerHarness = vi.hoisted(() => ({
  routes: [] as Array<Record<string, any>>,
}))

vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(() => ({})),
  createRouter: vi.fn((options: { routes: Array<Record<string, any>> }) => {
    routerHarness.routes = options.routes
    return {
      beforeEach: vi.fn(),
      afterEach: vi.fn(),
      onError: vi.fn(),
    }
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    checkAuth: vi.fn(),
    isAuthenticated: true,
    isAdmin: false,
    isSimpleMode: false,
    hasPendingAuthSession: false,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    siteName: 'Sub2API',
    backendModeEnabled: false,
    publicSettingsLoaded: true,
    cachedPublicSettings: null,
    fetchPublicSettings: vi.fn(),
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/stores/adminCompliance', () => ({
  useAdminComplianceStore: () => ({
    initialized: true,
    fetchStatus: vi.fn(),
    requireAcknowledgement: vi.fn(),
  }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({ triggerPrefetch: vi.fn() }),
}))

vi.mock('@/api/setup', () => ({
  getSetupStatus: vi.fn(),
}))

describe('fork extension routes', () => {
  beforeAll(async () => {
    await import('@/router')
  })

  it.each([
    ['/invoices', 'Invoices', false],
    ['/support-tickets', 'SupportTickets', false],
    ['/custom/:id', 'CustomPage', false],
    ['/admin/invoices', 'AdminInvoices', true],
    ['/admin/support-tickets', 'AdminSupportTickets', true],
  ])('keeps %s registered', (path, name, requiresAdmin) => {
    const route = routerHarness.routes.find((candidate) => candidate.path === path)

    expect(route).toBeDefined()
    expect(route?.name).toBe(name)
    expect(route?.meta?.requiresAuth).toBe(true)
    expect(route?.meta?.requiresAdmin).toBe(requiresAdmin)
    expect(typeof route?.component).toBe('function')
  })

  it('keeps the user support-ticket visibility guard metadata', () => {
    const route = routerHarness.routes.find((candidate) => candidate.path === '/support-tickets')
    expect(route?.meta?.requiresSupportTickets).toBe(true)
  })
})
