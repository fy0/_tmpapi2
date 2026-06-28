import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import HomeView from '../HomeView.vue'
import type { PublicSettings, User } from '@/types'

const stores = vi.hoisted(() => ({
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as User | null,
    checkAuth: vi.fn()
  },
  appStore: {
    cachedPublicSettings: null as PublicSettings | null,
    siteName: 'URPG API',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'home.dashboard': 'Dashboard',
  'home.login': 'Login',
  'home.getStarted': 'Get Started',
  'home.goToDashboard': 'Go to Dashboard',
  'home.viewDocs': 'Docs',
  'home.switchToLight': 'Light',
  'home.switchToDark': 'Dark',
  'home.tags.subscriptionToApi': 'Subscription to API',
  'home.tags.stickySession': 'Session Persistence',
  'home.tags.realtimeBilling': 'Pay As You Go',
  'home.mirrors.title': 'Access Mirrors',
  'home.mirrors.description': 'Choose a nearby route for the same API service',
  'home.mirrors.currentSite': 'Main Site',
  'home.mirrors.default': 'Default',
  'home.mirrors.action': 'Sign in to create and manage keys through this route',
  'home.customLinks.title': 'Featured Links',
  'home.customLinks.description': 'Common pages and external resources',
  'home.features.unifiedGateway': 'One-Click Access',
  'home.features.unifiedGatewayDesc': 'Get a single API key',
  'home.features.multiAccount': 'Always Reliable',
  'home.features.multiAccountDesc': 'Smart routing',
  'home.features.balanceQuota': 'Pay What You Use',
  'home.features.balanceQuotaDesc': 'Usage-based billing',
  'home.providers.title': 'Supported AI Models',
  'home.providers.description': 'One API, Multiple Choices',
  'home.providers.claude': 'Claude',
  'home.providers.gemini': 'Gemini',
  'home.providers.antigravity': 'Antigravity',
  'home.providers.more': 'More',
  'home.providers.supported': 'Supported',
  'home.providers.soon': 'Soon',
  'home.docs': 'Docs',
  'home.footer.allRightsReserved': 'All rights reserved.'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => stores.authStore,
  useAppStore: () => stores.appStore
}))

function publicSettings(overrides: Partial<PublicSettings> = {}): PublicSettings {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'URPG API',
    site_logo: '',
    site_subtitle: 'Stable API service',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    payment_enabled: false,
    risk_control_enabled: false,
    table_default_page_size: 20,
    table_page_size_options: [10, 20, 50, 100],
    custom_menu_items: [],
    custom_endpoints: [],
    custom_home_links: [],
    ticket_entry_visibility: 'all',
    linuxdo_oauth_enabled: false,
    wechat_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    version: '',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    channel_monitor_enabled: true,
    channel_monitor_default_interval_seconds: 60,
    available_channels_enabled: false,
    service_quota_enabled: false,
    affiliate_enabled: false,
    ...overrides
  }
}

function mountHome() {
  return mount(HomeView, {
    global: {
      stubs: {
        LocaleSwitcher: true,
        Icon: true,
        RouterLink: {
          props: ['to'],
          template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>'
        }
      }
    }
  })
}

describe('HomeView mirror cards', () => {
  beforeEach(() => {
    stores.authStore.isAuthenticated = false
    stores.authStore.isAdmin = false
    stores.authStore.user = null
    stores.authStore.checkAuth.mockReset()
    stores.appStore.fetchPublicSettings.mockReset()
    stores.appStore.publicSettingsLoaded = true
    stores.appStore.cachedPublicSettings = publicSettings()
    localStorage.clear()
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn().mockReturnValue({ matches: false })
    })
  })

  it('renders custom mirrors before the default mirror and links guests to login', () => {
    stores.appStore.cachedPublicSettings = publicSettings({
      api_base_url: 'https://api.example.com/',
      custom_endpoints: [
        {
          name: 'US Mirror',
          endpoint: 'https://us.example.com/',
          description: 'North America route'
        },
        {
          name: 'JP Mirror',
          endpoint: 'https://jp.example.com',
          description: 'Japan route'
        }
      ]
    })

    const wrapper = mountHome()
    const cards = wrapper.findAll('[data-testid="home-mirror-card"]')

    expect(cards).toHaveLength(3)
    expect(cards[0].text()).toContain('US Mirror')
    expect(cards[0].text()).toContain('https://us.example.com')
    expect(cards[0].text()).toContain('North America route')
    expect(cards[1].text()).toContain('JP Mirror')
    expect(cards[2].text()).toContain('Main Site')
    expect(cards[2].text()).toContain('Default')
    expect(cards[2].text()).toContain('https://api.example.com')
    expect(cards.every((card) => card.attributes('href') === '/login')).toBe(true)
  })

  it('uses the current site as the default mirror when no API base URL is configured', () => {
    stores.appStore.cachedPublicSettings = publicSettings({
      api_base_url: '',
      custom_endpoints: []
    })

    const wrapper = mountHome()
    const cards = wrapper.findAll('[data-testid="home-mirror-card"]')

    expect(cards).toHaveLength(1)
    expect(cards[0].text()).toContain('Main Site')
    expect(cards[0].text()).toContain(window.location.origin)
    expect(cards[0].text()).toContain('Sign in to create and manage keys through this route')
  })

  it('links authenticated admins to the admin dashboard', () => {
    stores.authStore.isAuthenticated = true
    stores.authStore.isAdmin = true
    stores.authStore.user = {
      id: 1,
      username: 'Admin',
      email: 'admin@example.com',
      role: 'admin',
      balance: 0,
      concurrency: 1,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: false,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    }
    stores.appStore.cachedPublicSettings = publicSettings({
      api_base_url: 'https://api.example.com'
    })

    const wrapper = mountHome()
    const card = wrapper.find('[data-testid="home-mirror-card"]')

    expect(card.attributes('href')).toBe('/admin/dashboard')
  })

  it('renders custom home links in a separate section', () => {
    stores.appStore.cachedPublicSettings = publicSettings({
      custom_home_links: [
        {
          id: 1,
          title: 'Docs',
          description: 'Read the guide',
          url: '/docs',
          open_in_new_window: false,
          sort_order: 20
        },
        {
          id: 2,
          title: 'Status',
          description: 'External status page',
          url: 'https://status.example.com',
          open_in_new_window: true,
          sort_order: 10
        }
      ]
    })

    const wrapper = mountHome()
    const links = wrapper.findAll('[data-testid="home-custom-link"]')

    expect(wrapper.text()).toContain('Featured Links')
    expect(links).toHaveLength(2)
    expect(links[0].text()).toContain('Status')
    expect(links[0].attributes('href')).toBe('https://status.example.com')
    expect(links[0].attributes('target')).toBe('_blank')
    expect(links[0].attributes('rel')).toBe('noopener noreferrer')
    expect(links[1].text()).toContain('Docs')
    expect(links[1].attributes('href')).toBe('/docs')
    expect(links[1].attributes('target')).toBeUndefined()
  })
})
