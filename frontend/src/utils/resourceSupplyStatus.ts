type TranslateFn = (key: string) => string

const knownSupplyStatuses = new Set([
  'none',
  'testing',
  'pending_review',
  'schedulable',
  'paused',
  'rejected',
  'revoked'
])

function normalizedSupplyStatus(status?: string | null): string {
  const value = String(status || 'none').trim()
  return value || 'none'
}

function supplyStatusTranslationKey(status: string, namespace: 'statuses' | 'statusDescriptions'): string {
  return `resourceSupply.${namespace}.${knownSupplyStatuses.has(status) ? status : 'unknown'}`
}

export function formatResourceSupplyStatus(status: string | null | undefined, t: TranslateFn): string {
  const normalized = normalizedSupplyStatus(status)
  if (!knownSupplyStatuses.has(normalized)) {
    return normalized
  }
  return t(supplyStatusTranslationKey(normalized, 'statuses'))
}

export function resourceSupplyStatusDescription(status: string | null | undefined, t: TranslateFn): string {
  return t(supplyStatusTranslationKey(normalizedSupplyStatus(status), 'statusDescriptions'))
}

export function resourceSupplyStatusBadgeClass(status: string | null | undefined): string {
  switch (normalizedSupplyStatus(status)) {
    case 'schedulable':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'pending_review':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400'
    case 'testing':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
    case 'paused':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-700/30 dark:text-gray-400'
    case 'rejected':
    case 'revoked':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700/30 dark:text-gray-400'
  }
}

export function formatResourceSupplySource(source: string | null | undefined, t: TranslateFn): string {
  switch (String(source || '').trim()) {
    case 'admin':
      return t('resourceSupply.sources.admin')
    case 'self_service':
      return t('resourceSupply.sources.self_service')
    default:
      return source || '-'
  }
}
