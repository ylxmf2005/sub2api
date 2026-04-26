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

      <div v-else class="space-y-6">
        <section v-for="summary in summaries" :key="summary.group.id" class="space-y-4">
          <div class="card p-4">
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

          <SettlementPoolOverview
            :summary="summary"
            :current-user-id="authStore.user?.id ?? null"
          />

          <section class="card p-4">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.participants') }}</h3>
            <DataTable
              :columns="participantColumns"
              :data="participantRows(summary)"
              row-key="user_id"
              :loading="loading"
            >
              <template #cell-user="{ row }">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.email }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  #{{ row.user_id }} <span v-if="row.username">{{ row.username }}</span>
                </div>
              </template>
              <template #cell-raw_usage="{ value }">
                <span class="tabular-nums">{{ money(value) }}</span>
              </template>
              <template #cell-weighted_usage="{ value }">
                <span class="tabular-nums">{{ money(value) }}</span>
              </template>
              <template #cell-current_tier="{ row }">
                <span class="tabular-nums">{{ currentTierLabel(summary, row.current_tier) }}</span>
              </template>
              <template #cell-fixed_share="{ value }">
                <span class="tabular-nums">{{ money(value) }}</span>
              </template>
              <template #cell-dynamic_charge="{ value }">
                <span class="tabular-nums">{{ money(value) }}</span>
              </template>
              <template #cell-total_due="{ value }">
                <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ money(value) }}</span>
              </template>
              <template #empty>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noActiveEstimate') }}</p>
              </template>
            </DataTable>
          </section>

          <section class="card p-4">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.cycles') }}</h3>
            <DataTable
              :columns="cycleColumns"
              :data="cycleRows(summary)"
              row-key="id"
              :loading="loading"
            >
              <template #cell-period="{ row }">
                <span class="text-gray-700 dark:text-gray-200">
                  {{ date(row.started_at) }} - {{ row.ended_at ? date(row.ended_at) : t('settlementPools.status.active') }}
                </span>
              </template>
              <template #cell-status="{ row }">
                <span class="badge" :class="row.status === 'active' ? 'badge-success' : 'badge-secondary'">
                  {{ t(`settlementPools.status.${row.status}`) }}
                </span>
              </template>
              <template #cell-total_due="{ row }">
                <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ money(cycleTotalDue(row)) }}</span>
              </template>
              <template #cell-actions="{ row }">
                <div class="flex justify-end">
                  <button
                    class="btn btn-secondary btn-sm"
                    :disabled="selectedCycleId(summary) === row.cycle_id"
                    @click="selectedCycleIds[summary.group.id] = row.cycle_id"
                  >
                    {{ t('common.view') }}
                  </button>
                </div>
              </template>
              <template #empty>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noCycles') }}</p>
              </template>
            </DataTable>
          </section>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DataTable from '@/components/common/DataTable.vue'
import SettlementPoolOverview from '@/components/settlement/SettlementPoolOverview.vue'
import settlementPoolsAPI from '@/api/settlementPools'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { Column } from '@/components/common/types'
import type { SettlementPoolCycle, SettlementPoolEstimate, SettlementPoolParticipantEstimate, SettlementPoolSummary, SettlementPoolTier } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const loading = ref(false)
const summaries = ref<SettlementPoolSummary[]>([])
const selectedCycleIds = ref<Record<number, number | null>>({})
const rightAlignedColumnClass = 'text-right [&>div]:justify-end'
const participantColumns = computed<Column[]>(() => [
  { key: 'user', label: t('settlementPools.user'), class: 'min-w-[220px]' },
  { key: 'raw_usage', label: t('settlementPools.rawUsage'), class: rightAlignedColumnClass },
  { key: 'weighted_usage', label: t('settlementPools.weightedUsage'), class: rightAlignedColumnClass },
  { key: 'current_tier', label: t('settlementPools.currentTier'), class: rightAlignedColumnClass },
  { key: 'fixed_share', label: t('settlementPools.fixedShare'), class: rightAlignedColumnClass },
  { key: 'dynamic_charge', label: t('settlementPools.dynamicCharge'), class: rightAlignedColumnClass },
  { key: 'total_due', label: t('settlementPools.totalDue'), class: rightAlignedColumnClass }
])
const cycleColumns = computed<Column[]>(() => [
  { key: 'period', label: t('settlementPools.period'), class: 'min-w-[260px]' },
  { key: 'status', label: t('common.status') },
  { key: 'total_due', label: t('settlementPools.totalDue'), class: rightAlignedColumnClass },
  { key: 'actions', label: '', class: rightAlignedColumnClass }
])

type CycleRow = {
  id: string
  cycle_id: number | null
  estimate: SettlementPoolEstimate | null
  group_id: number
  status: 'active' | 'locked'
  started_at: string
  ended_at?: string | null
  total_cost: number
  base_ratio: number
  market_cap: number
  tiers: SettlementPoolTier[]
}

function selectedCycleId(summary: SettlementPoolSummary): number | null {
  return selectedCycleIds.value[summary.group.id] ?? null
}

function displayEstimate(summary: SettlementPoolSummary): SettlementPoolEstimate | null {
  const cycleId = selectedCycleId(summary)
  if (cycleId != null) {
    return summaryCycles(summary).find(cycle => cycle.id === cycleId)?.snapshot ?? null
  }
  return summary.estimate ?? summaryCycles(summary).find(cycle => cycle.snapshot)?.snapshot ?? null
}

function summaryCycles(summary: SettlementPoolSummary): SettlementPoolCycle[] {
  return summary.cycles || []
}

function participantRows(summary: SettlementPoolSummary): SettlementPoolParticipantEstimate[] {
  return displayEstimate(summary)?.participants || []
}

function cycleRows(summary: SettlementPoolSummary): CycleRow[] {
  const rows: CycleRow[] = []
  if (summary.estimate) {
    rows.push({
      id: `active-${summary.group.id}`,
      cycle_id: null,
      estimate: summary.estimate,
      group_id: summary.group.id,
      status: summary.estimate.status,
      started_at: summary.estimate.started_at,
      ended_at: summary.estimate.ended_at,
      total_cost: summary.estimate.total_cost,
      base_ratio: summary.estimate.base_ratio,
      market_cap: summary.estimate.market_cap,
      tiers: summary.estimate.tiers
    })
  }
  return rows.concat(summaryCycles(summary).map(cycle => ({
    ...cycle,
    id: String(cycle.id),
    cycle_id: cycle.id,
    estimate: cycle.snapshot ?? null
  })))
}

function cycleTotalDue(row: CycleRow): number | null | undefined {
  return row.estimate?.participants.find(participant => participant.user_id === authStore.user?.id)?.total_due
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
