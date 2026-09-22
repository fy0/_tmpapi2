import type { CustomEndpoint, GroupPlatform } from '@/types'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.5'
export const GROK_CC_SWITCH_MODEL = 'grok-4.5'

export type CcSwitchClientType = 'claude' | 'gemini'

export interface CcSwitchMirrorOption {
  id: string
  name: string
  endpoint: string
  description: string
  isDefault: boolean
}

export interface CcSwitchMirrorOptionsInput {
  apiBaseUrl?: string | null
  customEndpoints?: CustomEndpoint[] | null
  fallbackBaseUrl?: string
  defaultName: string
}

export interface CcSwitchProviderNameInput {
  siteName?: string | null
  apiKeyName?: string | null
  mirrorName?: string | null
}

export interface CcSwitchImportConfig {
  app: string
  endpoint: string
  model?: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | null
  clientType: CcSwitchClientType
  providerName: string
  apiKey: string
  usageScript: string
}

function normalizeEndpoint(value: string | null | undefined): string {
  return (value || '').trim().replace(/\/+$/, '')
}

export function buildCcSwitchProviderName(input: CcSwitchProviderNameInput): string {
  const siteName = input.siteName?.trim() || 'sub2api'
  return [siteName, input.apiKeyName?.trim(), input.mirrorName?.trim()].filter(Boolean).join(' - ')
}

export function buildCcSwitchMirrorOptions(input: CcSwitchMirrorOptionsInput): CcSwitchMirrorOption[] {
  const options: CcSwitchMirrorOption[] = []
  const seen = new Set<string>()

  const addOption = (option: Omit<CcSwitchMirrorOption, 'id'>) => {
    const endpoint = normalizeEndpoint(option.endpoint)
    if (!endpoint || seen.has(endpoint)) return

    seen.add(endpoint)
    options.push({
      ...option,
      id: option.isDefault ? 'default' : `custom-${options.length}`,
      endpoint
    })
  }

  for (const item of input.customEndpoints || []) {
    addOption({
      name: item.name?.trim() || item.endpoint?.trim() || '',
      endpoint: item.endpoint,
      description: item.description?.trim() || '',
      isDefault: false
    })
  }

  addOption({
    name: input.defaultName,
    endpoint: normalizeEndpoint(input.apiBaseUrl) || normalizeEndpoint(input.fallbackBaseUrl),
    description: '',
    isDefault: true
  })

  return options
}

function withV1Endpoint(baseUrl: string): string {
  const normalizedBaseUrl = baseUrl.replace(/\/+$/, '')
  return normalizedBaseUrl.endsWith('/v1') ? normalizedBaseUrl : `${normalizedBaseUrl}/v1`
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType,
  baseUrl: string
): CcSwitchImportConfig {
  switch (platform || 'anthropic') {
    case 'antigravity':
      return {
        app: clientType === 'gemini' ? 'gemini' : 'claude',
        endpoint: `${baseUrl.replace(/\/+$/, '')}/antigravity`
      }
    case 'openai':
      return {
        app: 'codex',
        endpoint: withV1Endpoint(baseUrl),
        model: OPENAI_CC_SWITCH_CODEX_MODEL
      }
    case 'gemini':
      return {
        app: 'gemini',
        endpoint: baseUrl
      }
    case 'grok':
      return {
        app: 'grokbuild',
        endpoint: withV1Endpoint(baseUrl),
        model: GROK_CC_SWITCH_MODEL
      }
    default:
      return {
        app: 'claude',
        endpoint: baseUrl
      }
  }
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const config = resolveCcSwitchImportConfig(input.platform, input.clientType, input.baseUrl)
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', input.baseUrl],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(input.usageScript)],
    ['usageAutoInterval', '30']
  ]

  if (config.model) {
    entries.splice(2, 0, ['model', config.model])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
