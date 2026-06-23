import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'

import VersionBadge from '../VersionBadge.vue'
import { useAppStore } from '@/stores'

describe('VersionBadge', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders a static version badge from props', () => {
    const wrapper = mount(VersionBadge, {
      props: {
        version: 'v0.1.138',
      },
    })

    expect(wrapper.text()).toBe('v0.1.138')
    expect(wrapper.attributes('title')).toBe('v0.1.138')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('normalizes versions without duplicating the v prefix', () => {
    const wrapper = mount(VersionBadge, {
      props: {
        version: '0.1.138',
      },
    })

    expect(wrapper.text()).toBe('v0.1.138')
  })

  it('prefers the current version from the app store', () => {
    const appStore = useAppStore()
    appStore.currentVersion = 'v0.1.138'

    const wrapper = mount(VersionBadge, {
      props: {
        version: 'v0.0.1',
      },
    })

    expect(wrapper.text()).toBe('v0.1.138')
  })
})
