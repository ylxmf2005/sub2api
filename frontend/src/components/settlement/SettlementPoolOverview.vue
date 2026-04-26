<template>
  <section class="space-y-4">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.publicParameters') }}</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ periodLabel }}</p>
      </div>
      <span
        :class="[
          'badge',
          estimate?.status === 'active' ? 'badge-success' : 'badge-secondary'
        ]"
      >
        {{ t(`settlementPools.status.${estimate?.status || 'history'}`) }}
      </span>
    </div>

    <div v-if="estimate" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="card in overviewCards"
        :key="card.key"
        class="card p-4"
      >
        <div class="flex items-center gap-3">
          <div :class="['rounded-lg p-2', card.iconBgClass]">
            <Icon :name="card.icon" size="md" :class="card.iconClass" :stroke-width="2" />
          </div>
          <div class="min-w-0">
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</p>
            <p :class="['mt-1 break-words text-xl font-bold text-gray-900 dark:text-white', card.valueClass]">{{ card.value }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-if="estimate && myParticipant" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="card in personalCards"
        :key="card.key"
        class="card p-4"
      >
        <div class="flex items-center gap-3">
          <div :class="['rounded-lg p-2', card.iconBgClass]">
            <Icon :name="card.icon" size="md" :class="card.iconClass" :stroke-width="2" />
          </div>
          <div class="min-w-0">
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</p>
            <p :class="['mt-1 break-words text-xl font-bold text-gray-900 dark:text-white', card.valueClass]">{{ card.value }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-if="estimate">
      <div class="mb-2 flex items-center justify-between">
        <h4 class="text-sm font-medium text-gray-900 dark:text-white">{{ t('settlementPools.tiers') }}</h4>
        <span class="text-xs text-gray-500 dark:text-gray-400">{{ tiers.length }}</span>
      </div>
      <div class="grid gap-2 md:grid-cols-3">
        <div
          v-for="(tier, index) in tiers"
          :key="`${tier.up_to ?? 'open'}-${index}`"
          class="flex items-center justify-between rounded-md bg-gray-50 px-3 py-2 text-sm dark:bg-dark-700"
        >
          <span class="text-gray-600 dark:text-gray-300">{{ tierRangeLabel(tiers, index) }}</span>
          <span class="font-medium text-gray-900 dark:text-white">{{ tier.weight }}</span>
        </div>
      </div>
    </div>

    <div v-else class="mt-4 rounded-lg border border-dashed border-gray-300 px-4 py-8 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
      {{ t('settlementPools.noActiveEstimate') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { SettlementPoolEstimate, SettlementPoolSummary, SettlementPoolTier } from '@/types'

type OverviewIcon = 'bolt' | 'chart' | 'creditCard' | 'database' | 'exclamationTriangle' | 'grid' | 'shield' | 'trendingUp' | 'userCircle' | 'users'
type OverviewCard = {
  key: string
  label: string
  value: string
  icon: OverviewIcon
  iconBgClass: string
  iconClass: string
  valueClass?: string
}

const props = defineProps<{
  summary: SettlementPoolSummary
  currentUserId?: number | null
  displayEstimate?: SettlementPoolEstimate | null
}>()

const { t } = useI18n()

const estimate = computed(() => (
  props.displayEstimate !== undefined
    ? props.displayEstimate
    : props.summary.estimate ?? props.summary.cycles.find(cycle => cycle.snapshot)?.snapshot ?? null
))
const myParticipant = computed(() => {
  if (!props.currentUserId || !estimate.value) return null
  return estimate.value.participants.find(row => row.user_id === props.currentUserId) ?? null
})
const totalCost = computed(() => estimate.value?.total_cost ?? props.summary.active_cycle?.total_cost ?? 0)
const baseRatio = computed(() => estimate.value?.base_ratio ?? props.summary.active_cycle?.base_ratio ?? props.summary.config?.base_ratio ?? 0)
const marketCap = computed(() => estimate.value?.market_cap ?? props.summary.active_cycle?.market_cap ?? props.summary.config?.market_cap ?? 0)
const tiers = computed<SettlementPoolTier[]>(() => estimate.value?.tiers ?? props.summary.active_cycle?.tiers ?? props.summary.config?.tiers ?? [])
const periodLabel = computed(() => {
  if (!estimate.value) return t('settlementPools.noActiveEstimate')
  return `${formatDate(estimate.value.started_at)} - ${estimate.value.ended_at ? formatDate(estimate.value.ended_at) : t('settlementPools.status.active')}`
})
const overviewCards = computed<OverviewCard[]>(() => [
  {
    key: 'total_cost',
    label: t('settlementPools.totalCostShort'),
    value: cnyMoney(totalCost.value),
    icon: 'database',
    iconBgClass: 'bg-primary-100 dark:bg-primary-900/20',
    iconClass: 'text-primary-600 dark:text-primary-400'
  },
  {
    key: 'base_ratio',
    label: t('settlementPools.baseRatio'),
    value: percent(baseRatio.value),
    icon: 'chart',
    iconBgClass: 'bg-blue-100 dark:bg-blue-900/20',
    iconClass: 'text-blue-600 dark:text-blue-400'
  },
  {
    key: 'market_cap',
    label: t('settlementPools.marketCap'),
    value: cnyMoney(marketCap.value),
    icon: 'shield',
    iconBgClass: 'bg-emerald-100 dark:bg-emerald-900/20',
    iconClass: 'text-emerald-600 dark:text-emerald-400'
  },
  {
    key: 'uncapped_dynamic_rate',
    label: t('settlementPools.uncappedDynamicRate'),
    value: cnyMoney(estimate.value?.uncapped_dynamic_rate),
    icon: 'trendingUp',
    iconBgClass: 'bg-purple-100 dark:bg-purple-900/20',
    iconClass: 'text-purple-600 dark:text-purple-400'
  },
  {
    key: 'dynamic_rate',
    label: t('settlementPools.dynamicRate'),
    value: cnyMoney(estimate.value?.effective_dynamic_rate),
    icon: 'bolt',
    iconBgClass: 'bg-amber-100 dark:bg-amber-900/20',
    iconClass: 'text-amber-600 dark:text-amber-400'
  },
  {
    key: 'participants',
    label: t('settlementPools.participants'),
    value: String(estimate.value?.participant_count ?? 0),
    icon: 'users',
    iconBgClass: 'bg-cyan-100 dark:bg-cyan-900/20',
    iconClass: 'text-cyan-600 dark:text-cyan-400'
  },
  {
    key: 'owner_loss',
    label: t('settlementPools.ownerLoss'),
    value: cnyMoney(estimate.value?.owner_covered_loss),
    icon: 'exclamationTriangle',
    iconBgClass: 'bg-orange-100 dark:bg-orange-900/20',
    iconClass: 'text-orange-600 dark:text-orange-400',
    valueClass: 'text-orange-600 dark:text-orange-400'
  },
  {
    key: 'tiers',
    label: t('settlementPools.tiers'),
    value: String(tiers.value.length),
    icon: 'grid',
    iconBgClass: 'bg-gray-100 dark:bg-dark-700',
    iconClass: 'text-gray-600 dark:text-gray-300'
  }
])
const personalCards = computed<OverviewCard[]>(() => {
  if (!myParticipant.value) return []
  return [
    {
      key: 'my_due',
      label: t('settlementPools.myDue'),
      value: cnyMoney(myParticipant.value.total_due),
      icon: 'creditCard',
      iconBgClass: 'bg-primary-100 dark:bg-primary-900/20',
      iconClass: 'text-primary-600 dark:text-primary-400',
      valueClass: 'text-primary-700 dark:text-primary-300'
    },
    {
      key: 'my_fixed_share',
      label: t('settlementPools.myFixedShare'),
      value: cnyMoney(myParticipant.value.fixed_share),
      icon: 'shield',
      iconBgClass: 'bg-gray-100 dark:bg-dark-700',
      iconClass: 'text-gray-600 dark:text-gray-300'
    },
    {
      key: 'my_dynamic_charge',
      label: t('settlementPools.myDynamicCharge'),
      value: cnyMoney(myParticipant.value.dynamic_charge),
      icon: 'bolt',
      iconBgClass: 'bg-amber-100 dark:bg-amber-900/20',
      iconClass: 'text-amber-600 dark:text-amber-400'
    },
    {
      key: 'my_usage',
      label: t('settlementPools.myUsage'),
      value: usdMoney(myParticipant.value.raw_usage),
      icon: 'userCircle',
      iconBgClass: 'bg-cyan-100 dark:bg-cyan-900/20',
      iconClass: 'text-cyan-600 dark:text-cyan-400'
    }
  ]
})
function usdMoney(value: number | null | undefined) {
  return `$${Number(value || 0).toFixed(4)}`
}

function cnyMoney(value: number | null | undefined) {
  return `¥${Number(value || 0).toFixed(4)}`
}

function percent(value: number | null | undefined) {
  return `${(Number(value || 0) * 100).toFixed(2)}%`
}

function formatDate(value?: string | null) {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

function tierRangeLabel(currentTiers: SettlementPoolTier[], index: number) {
  const previous = index === 0 ? 0 : currentTiers[index - 1]?.up_to
  const current = currentTiers[index]?.up_to
  if (current == null) return `${previous ?? 0}+`
  return `${previous ?? 0}-${current}`
}
</script>
