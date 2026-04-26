<template>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
    <section class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="mb-4 flex items-start justify-between gap-3">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.currentEstimate') }}</h3>
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

      <div v-if="estimate" class="space-y-3">
        <div
          v-if="myParticipant"
          class="rounded-lg bg-primary-50 px-3 py-3 dark:bg-primary-900/10"
        >
          <p class="text-xs text-primary-700/80 dark:text-primary-300/80">{{ t('settlementPools.myDue') }}</p>
          <p class="mt-1 text-lg font-semibold text-primary-700 dark:text-primary-300">{{ money(myParticipant.total_due) }}</p>
        </div>
        <div
          v-if="myParticipant"
          class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.myFixedShare') }}</p>
          <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(myParticipant.fixed_share) }}</p>
        </div>
        <div
          v-if="myParticipant"
          class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.myDynamicCharge') }}</p>
          <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(myParticipant.dynamic_charge) }}</p>
        </div>
        <div
          v-if="myParticipant"
          class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.myUsage') }}</p>
          <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(myParticipant.raw_usage) }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ tierText(myParticipant.current_tier) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.totalCost') }}</p>
          <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(estimate.total_cost) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.dynamicRate') }}</p>
          <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(estimate.effective_dynamic_rate) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.participants') }}</p>
          <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ estimate.participant_count }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.ownerLoss') }}</p>
          <p class="mt-1 text-lg font-semibold text-amber-600 dark:text-amber-400">{{ money(estimate.owner_covered_loss) }}</p>
        </div>
      </div>

      <div v-else class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
        {{ t('settlementPools.noActiveEstimate') }}
      </div>
    </section>

    <section class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.publicParameters') }}</h3>
      <div class="mt-4 space-y-3">
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.totalCostShort') }}</p>
          <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ money(totalCost) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.baseRatio') }}</p>
          <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ percent(baseRatio) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.marketCap') }}</p>
          <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ money(marketCap) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.uncappedDynamicRate') }}</p>
          <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ money(estimate?.uncapped_dynamic_rate) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.fixedPool') }}</p>
          <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ money(estimate?.fixed_pool) }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('settlementPools.dynamicPool') }}</p>
          <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ money(estimate?.dynamic_pool) }}</p>
        </div>
      </div>

      <div class="mt-4">
        <div class="mb-2 flex items-center justify-between">
          <h4 class="text-sm font-medium text-gray-900 dark:text-white">{{ t('settlementPools.tiers') }}</h4>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ tiers.length }}</span>
        </div>
        <div class="space-y-2">
          <div
            v-for="(tier, index) in tiers"
            :key="`${tier.up_to ?? 'open'}-${index}`"
            class="flex items-center justify-between rounded-md bg-gray-50 px-3 py-2 text-sm dark:bg-dark-700"
          >
            <span class="text-gray-600 dark:text-gray-300">{{ tierText(index) }}</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ tier.weight }}</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SettlementPoolSummary, SettlementPoolTier } from '@/types'

const props = defineProps<{
  summary: SettlementPoolSummary
  currentUserId?: number | null
}>()

const { t } = useI18n()

const estimate = computed(() => props.summary.estimate ?? props.summary.cycles.find(cycle => cycle.snapshot)?.snapshot ?? null)
const myParticipant = computed(() => {
  if (!props.currentUserId || !estimate.value) return null
  return estimate.value.participants.find(row => row.user_id === props.currentUserId) ?? null
})
const totalCost = computed(() => props.summary.active_cycle?.total_cost ?? estimate.value?.total_cost ?? 0)
const baseRatio = computed(() => props.summary.active_cycle?.base_ratio ?? props.summary.config?.base_ratio ?? estimate.value?.base_ratio ?? 0)
const marketCap = computed(() => props.summary.active_cycle?.market_cap ?? props.summary.config?.market_cap ?? estimate.value?.market_cap ?? 0)
const tiers = computed<SettlementPoolTier[]>(() => props.summary.active_cycle?.tiers ?? props.summary.config?.tiers ?? estimate.value?.tiers ?? [])
const periodLabel = computed(() => {
  if (!estimate.value) return t('settlementPools.noActiveEstimate')
  return `${formatDate(estimate.value.started_at)} - ${estimate.value.ended_at ? formatDate(estimate.value.ended_at) : t('settlementPools.status.active')}`
})

function money(value: number | null | undefined) {
  return `$${Number(value || 0).toFixed(4)}`
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

function tierText(index: number) {
  const tier = tiers.value[index]
  if (!tier) return ''
  return `${tierRangeLabel(tiers.value, index)} x ${tier.weight}`
}
</script>
