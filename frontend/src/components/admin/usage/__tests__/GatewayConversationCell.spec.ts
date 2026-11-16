import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter, RouterLink } from 'vue-router'
import GatewayConversationCell from '../GatewayConversationCell.vue'
import type { GatewayRequestMetadata } from '@/types'

const { showSuccess, showError, writeText } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn(), writeText: vi.fn() }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

async function render(row: GatewayRequestMetadata & { user_id?: number | null; api_key_id?: number | null }) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/admin/usage', component: { template: '<div />' } }] })
  await router.push('/admin/usage?tab=errors')
  await router.isReady()
  return mount(GatewayConversationCell, { props: { row }, global: { plugins: [router] } })
}

beforeEach(() => {
  vi.clearAllMocks()
  writeText.mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
})

describe('GatewayConversationCell', () => {
  it('keeps unknown historical IDs unknown', async () => {
    const wrapper = await render({})
    expect(wrapper.get('[data-testid="conversation-unknown"]').text()).toBe('—')
    expect(wrapper.findComponent(RouterLink).exists()).toBe(false)
    expect(wrapper.find('[data-testid="conversation-copy"]').exists()).toBe(false)
  })

  it('scopes a same-conversation filter by both user and API key', async () => {
    const wrapper = await render({ conversation_id: 'same-id', conversation_id_source: 'header.conversation_id', user_id: 42, api_key_id: 7 })
    expect(wrapper.getComponent(RouterLink).props('to')).toEqual({ path: '/admin/usage', query: { tab: 'errors', conversation_id: 'same-id', conversation_id_source: 'header.conversation_id', user_id: '42', api_key_id: '7' } })
  })

  it('does not offer an unscoped identity link', async () => {
    const wrapper = await render({ conversation_id: 'same-id', user_id: 42 })
    expect(wrapper.findComponent(RouterLink).exists()).toBe(false)
    expect(wrapper.text()).toContain('same-id')
  })

  it('copies only the identifier', async () => {
    const wrapper = await render({ conversation_id: 'copy-me', conversation_id_source: 'fallback.prompt_cache_key' })
    await wrapper.get('[data-testid="conversation-copy"]').trigger('click')
    expect(writeText).toHaveBeenCalledWith('copy-me')
    expect(showSuccess).toHaveBeenCalled()
    expect(wrapper.text()).toContain('admin.usage.correlationFallback')
  })

  it('renders identifier markup as text and marks unknown attempt costs', async () => {
    const wrapper = await render({ conversation_id: '<script>no()</script>', attempt_count: 1, attempts: [{ attempt: 1, account_id: 87, upstream_status: 499, outcome: 'retrying', started_at: '2026-11-10T00:00:00Z', ended_at: '2026-11-10T00:00:01Z', cost_accounting: 'unknown' }] })
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.text()).toContain('<script>no()</script>')
    expect(wrapper.text()).toContain('499')
    expect(wrapper.text()).toContain('admin.usage.attemptCostUnknown')
    expect(wrapper.text()).toContain('admin.usage.correlationNotice')
  })
})
