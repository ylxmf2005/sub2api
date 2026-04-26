<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('settlementPools.userTitle') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.userDescription') }}</p>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <EmptyState
        v-else-if="summaries.length === 0"
        :title="t('settlementPools.noJoinedPools')"
        :description="t('settlementPools.noJoinedPoolsDesc')"
      />

      <template v-else>
      <section v-for="summary in summaries" :key="summary.group.id" class="card overflow-hidden">
        <div class="border-b border-gray-200 p-4 dark:border-dark-600">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div class="flex items-center gap-2">
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ summary.group.name }}</h2>
                <span :class="['badge', displayEstimate(summary)?.status === 'active' ? 'badge-success' : 'badge-secondary']">
                  {{ t(`settlementPools.status.${displayEstimate(summary)?.status === 'active' ? 'active' : 'history'}`) }}
                </span>
              </div>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ date(displayEstimate(summary)?.started_at) }}</p>
            </div>
          </div>
        </div>

        <div class="grid gap-6 p-4 xl:grid-cols-[1fr_320px]">
          <div class="space-y-6">
            <SettlementPoolOverview
              :summary="summary"
              :current-user-id="authStore.user?.id ?? null"
            />

            <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
              <thead class="bg-gray-50 dark:bg-dark-700">
                <tr>
                  <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.user') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.rawUsage') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.weightedUsage') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.currentTier') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.fixedShare') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.dynamicCharge') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.totalDue') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
                <tr
                  v-for="row in displayEstimate(summary)?.participants || []"
                  :key="row.user_id"
                  :class="row.user_id === authStore.user?.id ? 'bg-primary-50/70 dark:bg-primary-900/10' : ''"
                >
                  <td class="px-3 py-2">
                    <div class="font-medium text-gray-900 dark:text-white">{{ row.email }}</div>
                    <div class="text-xs text-gray-500">#{{ row.user_id }}</div>
                  </td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ money(row.raw_usage) }}</td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ money(row.weighted_usage) }}</td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ currentTierLabel(summary, row.current_tier) }}</td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ money(row.fixed_share) }}</td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ money(row.dynamic_charge) }}</td>
                  <td class="px-3 py-2 text-right font-medium tabular-nums text-gray-900 dark:text-white">{{ money(row.total_due) }}</td>
                </tr>
                <tr v-if="!displayEstimate(summary)">
                  <td colspan="7" class="px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noActiveEstimate') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          </div>

          <aside class="space-y-4">
            <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
              <h3 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">{{ t('settlementPools.cycles') }}</h3>
              <div class="space-y-2">
                <button
                  v-if="summary.estimate"
                  class="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-gray-50 dark:hover:bg-dark-700"
                  :class="selectedCycleIds[summary.group.id] == null ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300' : ''"
                  @click="selectedCycleIds[summary.group.id] = null"
                >
                  <span>{{ date(summary.estimate.started_at) }}</span>
                  <span class="font-medium">{{ money(summary.estimate.participants.find(row => row.user_id === authStore.user?.id)?.total_due) }}</span>
                </button>
                <button
                  v-for="cycle in summary.cycles"
                  :key="cycle.id"
                  class="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-gray-50 dark:hover:bg-dark-700"
                  :class="selectedCycleIds[summary.group.id] === cycle.id ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300' : 'text-gray-600 dark:text-gray-300'"
                  @click="selectedCycleIds[summary.group.id] = cycle.id"
                >
                  <span>{{ date(cycle.started_at) }}</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ money(cycle.snapshot?.participants?.find(row => row.user_id === authStore.user?.id)?.total_due) }}</span>
                </button>
                <p v-if="summary.cycles.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noCycles') }}</p>
              </div>
            </div>
          </aside>
        </div>
      </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import SettlementPoolOverview from '@/components/settlement/SettlementPoolOverview.vue'
import settlementPoolsAPI from '@/api/settlementPools'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { SettlementPoolEstimate, SettlementPoolSummary, SettlementPoolTier } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const loading = ref(false)
const summaries = ref<SettlementPoolSummary[]>([])
const selectedCycleIds = ref<Record<number, number | null>>({})

function displayEstimate(summary: SettlementPoolSummary): SettlementPoolEstimate | null {
  const selectedCycleId = selectedCycleIds.value[summary.group.id]
  if (selectedCycleId != null) {
    return summary.cycles.find(cycle => cycle.id === selectedCycleId)?.snapshot ?? null
  }
  return summary.estimate ?? summary.cycles.find(cycle => cycle.snapshot)?.snapshot ?? null
}

function money(value: number | null | undefined) {
  return `$${Number(value || 0).toFixed(4)}`
}

function date(value?: string | null) {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

function tierLabel(tiers: SettlementPoolTier[], index: number) {
  const previous = index === 0 ? 0 : tiers[index - 1]?.up_to
  const current = tiers[index]?.up_to
  if (current == null) return `${previous ?? 0}+`
  return `${previous ?? 0}-${current}`
}

function currentTierLabel(summary: SettlementPoolSummary, index: number) {
  const tiers = displayEstimate(summary)?.tiers || []
  const tier = tiers[index]
  if (!tier) return ''
  return `${tierLabel(tiers, index)} x ${tier.weight}`
}

async function load() {
  loading.value = true
  try {
    summaries.value = await settlementPoolsAPI.list()
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToLoad'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
