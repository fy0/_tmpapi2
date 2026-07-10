type LocaleRecord = Record<string, unknown>

function isRecord(value: unknown): value is LocaleRecord {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function mergeLocale<T extends LocaleRecord>(base: T, overlay: LocaleRecord): T {
  const merged: LocaleRecord = { ...base }
  for (const [key, value] of Object.entries(overlay)) {
    const current = merged[key]
    merged[key] = isRecord(current) && isRecord(value) ? mergeLocale(current, value) : value
  }
  return merged as T
}
