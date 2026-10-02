import { describe, expect, it } from 'vitest'
import {
  CC_SWITCH_USAGE_SCRIPT,
  GROK_CC_SWITCH_MODEL,
  OPENAI_CC_SWITCH_CODEX_MODEL,
  buildCcSwitchImportDeeplink,
  buildCcSwitchMirrorOptions,
  buildCcSwitchProviderName
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

  it('defaults Grok Build imports to the current Grok model', () => {
    expect(GROK_CC_SWITCH_MODEL).toBe('grok-4.5')
  })

  it('builds stable mirror options with custom endpoints before the default endpoint', () => {
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
        id: 'custom-0',
        name: '优化线路(位于北美)',
        endpoint: 'https://code-us.example.com',
        description: 'North America',
        isDefault: false
      },
      {
        id: 'custom-1',
        name: '优化线路(位于日本)',
        endpoint: 'https://code-jp.example.com',
        description: 'Japan',
        isDefault: false
      },
      {
        id: 'default',
        name: '主站',
        endpoint: 'https://api.example.com',
        description: '',
        isDefault: true
      }
    ])
  })

  it('lets custom endpoints override the default endpoint when URLs duplicate', () => {
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
        id: 'custom-0',
        name: 'Duplicate',
        endpoint: 'https://current.example.com',
        description: 'duplicate',
        isDefault: false
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

  it('builds distinguishable provider names from the site, API key, and mirror', () => {
    const usName = buildCcSwitchProviderName({
      siteName: 'Sub2API',
      apiKeyName: 'production',
      mirrorName: '优化线路(位于北美)'
    })
    const japanName = buildCcSwitchProviderName({
      siteName: 'Sub2API',
      apiKeyName: 'production',
      mirrorName: '优化线路(位于日本)'
    })

    expect(usName).toBe('Sub2API - production - 优化线路(位于北美)')
    expect(japanName).toBe('Sub2API - production - 优化线路(位于日本)')
    expect(usName).not.toBe(japanName)
  })

  it('trims provider name segments, falls back the site name, and skips empty segments', () => {
    expect(
      buildCcSwitchProviderName({
        siteName: '   ',
        apiKeyName: '  test key  ',
        mirrorName: ''
      })
    ).toBe('sub2api - test key')
  })

  const baseInput = {
    baseUrl: 'https://api.example.com',
    providerName: 'Sub2API',
    apiKey: 'sk-test',
    usageScript: 'return true'
  }

  it.each([
    ['https://api.example.com', 'https://api.example.com'],
    ['https://api.example.com/', 'https://api.example.com'],
    ['https://api.example.com/v1', 'https://api.example.com/v1'],
    ['https://api.example.com/v1/', 'https://api.example.com/v1']
  ])('keeps Codex imports on the configured endpoint for base URL %s', (baseUrl, endpoint) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        baseUrl,
        platform: 'openai',
        clientType: 'claude'
      })
    )

    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe(endpoint)
    expect(params.get('model')).toBe(OPENAI_CC_SWITCH_CODEX_MODEL)
    expect(atob(params.get('usageScript') || '')).toBe(baseInput.usageScript)
  })

  it.each([
    'https://api.example.com',
    'https://api.example.com/',
    'https://api.example.com/v1',
    'https://api.example.com/v1/'
  ])('imports Grok Build with one /v1 suffix for base URL %s', (baseUrl) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        baseUrl,
        platform: 'grok',
        clientType: 'claude'
      })
    )

    expect(params.get('app')).toBe('grokbuild')
    expect(params.get('endpoint')).toBe('https://api.example.com/v1')
    expect(params.get('model')).toBe(GROK_CC_SWITCH_MODEL)
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

  it('preserves the generated provider name in the import deeplink', () => {
    const providerName = buildCcSwitchProviderName({
      siteName: 'Sub2API',
      apiKeyName: 'production',
      mirrorName: '优化线路(位于日本)'
    })
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        providerName,
        platform: 'openai',
        clientType: 'claude'
      })
    )

    expect(params.get('name')).toBe('Sub2API - production - 优化线路(位于日本)')
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

describe('CC Switch usage script', () => {
  // Mirrors CC Switch: substitute the template vars as text, evaluate, read request.url.
  function usageUrlFor(baseUrl: string): string {
    const script = CC_SWITCH_USAGE_SCRIPT.split('{{baseUrl}}').join(baseUrl).split('{{apiKey}}').join('sk-test')
    // eslint-disable-next-line no-new-func
    const config = new Function(`return ${script}`)() as { request: { url: string } }
    return config.request.url
  }

  it.each([
    'https://api.example.com',
    'https://api.example.com/',
    'https://api.example.com/v1',
    'https://api.example.com/v1/'
  ])('queries exactly one /v1/usage for base URL %s', (baseUrl) => {
    expect(usageUrlFor(baseUrl)).toBe('https://api.example.com/v1/usage')
  })

  it('works against the endpoint every platform import stores', () => {
    for (const platform of ['anthropic', 'openai', 'grok', 'gemini'] as GroupPlatform[]) {
      const endpoint = paramsFromDeeplink(
        buildCcSwitchImportDeeplink({
          baseUrl: 'https://api.example.com',
          platform,
          clientType: platform === 'gemini' ? 'gemini' : 'claude',
          providerName: 'Sub2API',
          apiKey: 'sk-test',
          usageScript: CC_SWITCH_USAGE_SCRIPT
        })
      ).get('endpoint') as string
      expect(usageUrlFor(endpoint)).toBe('https://api.example.com/v1/usage')
    }
  })
})
