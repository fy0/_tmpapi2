import { describe, expect, it } from 'vitest'
import {
  OPENAI_CC_SWITCH_CODEX_MODEL,
  buildCcSwitchImportDeeplink,
  buildCcSwitchMirrorOptions
} from '@/utils/ccswitchImport'
import type { GroupPlatform } from '@/types'

function paramsFromDeeplink(deeplink: string): URLSearchParams {
  const query = deeplink.split('?')[1] || ''
  return new URLSearchParams(query)
}

describe('ccswitchImport utils', () => {
  it('defaults OpenAI CC Switch imports to the current Codex model', () => {
    expect(OPENAI_CC_SWITCH_CODEX_MODEL).toBe('gpt-5.5')
  })

  it('builds stable mirror options from the default endpoint and custom endpoints', () => {
    const options = buildCcSwitchMirrorOptions({
      apiBaseUrl: 'https://api.example.com/',
      fallbackBaseUrl: 'https://fallback.example.com',
      defaultName: '主站',
      customEndpoints: [
        {
          name: '优化线路(位于北美)',
          endpoint: 'https://code-us.example.com/',
          description: 'North America'
        },
        {
          name: '优化线路(位于日本)',
          endpoint: 'https://code-jp.example.com',
          description: 'Japan'
        }
      ]
    })

    expect(options).toEqual([
      {
        id: 'default',
        name: '主站',
        endpoint: 'https://api.example.com',
        description: '',
        isDefault: true
      },
      {
        id: 'custom-1',
        name: '优化线路(位于北美)',
        endpoint: 'https://code-us.example.com',
        description: 'North America',
        isDefault: false
      },
      {
        id: 'custom-2',
        name: '优化线路(位于日本)',
        endpoint: 'https://code-jp.example.com',
        description: 'Japan',
        isDefault: false
      }
    ])
  })

  it('falls back to the current origin and skips empty or duplicated mirror endpoints', () => {
    const options = buildCcSwitchMirrorOptions({
      apiBaseUrl: '',
      fallbackBaseUrl: 'https://current.example.com/',
      defaultName: 'Default',
      customEndpoints: [
        { name: 'Empty', endpoint: '', description: '' },
        { name: 'Duplicate', endpoint: 'https://current.example.com', description: 'duplicate' },
        { name: '', endpoint: 'https://backup.example.com/path/', description: '  backup  ' }
      ]
    })

    expect(options).toEqual([
      {
        id: 'default',
        name: 'Default',
        endpoint: 'https://current.example.com',
        description: '',
        isDefault: true
      },
      {
        id: 'custom-1',
        name: 'https://backup.example.com/path/',
        endpoint: 'https://backup.example.com/path',
        description: 'backup',
        isDefault: false
      }
    ])
  })

  const baseInput = {
    baseUrl: 'https://api.example.com',
    providerName: 'Sub2API',
    apiKey: 'sk-test',
    usageScript: 'return true'
  }

  it('adds the Codex model parameter for OpenAI imports', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'openai',
        clientType: 'claude'
      })
    )

    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe(baseInput.baseUrl)
    expect(params.get('model')).toBe(OPENAI_CC_SWITCH_CODEX_MODEL)
    expect(atob(params.get('usageScript') || '')).toBe(baseInput.usageScript)
  })

  it.each([
    { platform: 'anthropic' as GroupPlatform, clientType: 'claude' as const, app: 'claude' },
    { platform: 'gemini' as GroupPlatform, clientType: 'gemini' as const, app: 'gemini' }
  ])('does not add a model parameter for $platform imports', ({ platform, clientType, app }) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform,
        clientType
      })
    )

    expect(params.get('app')).toBe(app)
    expect(params.get('endpoint')).toBe(baseInput.baseUrl)
    expect(params.has('model')).toBe(false)
  })

  it('keeps Antigravity imports on the selected client endpoint without a model parameter', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'antigravity',
        clientType: 'gemini'
      })
    )

    expect(params.get('app')).toBe('gemini')
    expect(params.get('endpoint')).toBe(`${baseInput.baseUrl}/antigravity`)
    expect(params.has('model')).toBe(false)
  })

  it('uses the selected mirror endpoint when building OpenAI imports', () => {
    const selectedMirror = 'https://code-us.example.com'
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        baseUrl: selectedMirror,
        platform: 'openai',
        clientType: 'claude'
      })
    )

    expect(params.get('homepage')).toBe(selectedMirror)
    expect(params.get('endpoint')).toBe(selectedMirror)
    expect(params.get('model')).toBe(OPENAI_CC_SWITCH_CODEX_MODEL)
  })

  it('uses the selected mirror endpoint for Antigravity client imports', () => {
    const selectedMirror = 'https://code-jp.example.com'
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        baseUrl: selectedMirror,
        platform: 'antigravity',
        clientType: 'gemini'
      })
    )

    expect(params.get('homepage')).toBe(selectedMirror)
    expect(params.get('endpoint')).toBe(`${selectedMirror}/antigravity`)
    expect(params.get('app')).toBe('gemini')
  })
})
