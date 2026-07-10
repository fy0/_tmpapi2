import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const zhLocalePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../i18n/locales/zh/local.ts')
const enLocalePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../i18n/locales/en/local.ts')
const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')
const enLocaleSource = readFileSync(enLocalePath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

function sourceBetween(start: string, end: string): string {
  const startIndex = componentSource.indexOf(start)
  const endIndex = componentSource.indexOf(end, startIndex + start.length)

  expect(startIndex).toBeGreaterThanOrEqual(0)
  expect(endIndex).toBeGreaterThan(startIndex)

  return componentSource.slice(startIndex, endIndex)
}

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar version badge', () => {
  it('uses the static version badge without dropdown behavior', () => {
    expect(componentSource).toContain('<VersionBadge :version="siteVersion" />')
    expect(componentSource).not.toContain('version badge dropdown')
    expect(componentSource).not.toContain('check-updates')
  })
})

describe('AppSidebar user navigation sections', () => {
  it('defines localized section titles for API management and services', () => {
    expect(componentSource).toContain("title: t('nav.apiManagement')")
    expect(componentSource).toContain("title: t('nav.services')")
    expect(zhLocaleSource).toContain('"apiManagement": "API 管理"')
    expect(zhLocaleSource).toContain('"services": "服务"')
    expect(enLocaleSource).toContain('"apiManagement": "API Management"')
    expect(enLocaleSource).toContain('"services": "Services"')
  })

  it('groups API-related user routes under API management', () => {
    const apiManagementSection = sourceBetween("key: 'api-management'", "key: 'services'")

    expect(apiManagementSection).toContain("path: '/keys'")
    expect(apiManagementSection).toContain("path: '/batch-image'")
    expect(apiManagementSection).toContain("path: '/usage'")
    expect(apiManagementSection).toContain("path: '/available-channels'")
    expect(apiManagementSection).toContain("path: '/monitor'")
  })

  it('groups service routes under services and keeps profile out of the sidebar', () => {
    const servicesSection = sourceBetween("key: 'services'", "key: 'custom-default'")
    const defaultCustomSection = sourceBetween("key: 'custom-default'", '\n  return sections')

    expect(servicesSection).toContain("path: '/subscriptions'")
    expect(servicesSection).toContain("path: '/invoices'")
    expect(servicesSection).toContain("path: '/support-tickets'")
    expect(servicesSection).toContain("path: '/redeem'")
    expect(defaultCustomSection).not.toContain("path: '/profile'")
    expect(defaultCustomSection).toContain('defaultCustomMenuItemsForUser.value.map(customMenuNavItem)')
  })

  it('merges custom groups with built-in sections before rendering standalone groups', () => {
    expect(componentSource).toContain('mergeCustomGroupSectionsIntoNavSections(')
    expect(componentSource).toContain('targetSection.items.push(...customSection.items)')
    expect(componentSource).toContain('standaloneCustomMenuGroupSectionsForUser')
    expect(componentSource).toContain('standaloneCustomMenuGroupSectionsForAdminPersonal')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('saves and restores scroll position', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})
