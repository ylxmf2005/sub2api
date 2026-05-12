<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('settlementPools.adminTitle') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.adminDescription') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="refresh">
          <Icon name="refresh" size="sm" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="loading && !summary" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <EmptyState
        v-else-if="settlementGroups.length === 0"
        :title="t('settlementPools.noGroups')"
        :description="t('settlementPools.noGroupsDesc')"
        :action-text="t('admin.groups.createGroup')"
        action-to="/admin/groups"
      />

      <template v-else>
        <div class="card p-4">
          <div class="grid gap-4 lg:grid-cols-[minmax(240px,360px)_auto] lg:items-end lg:justify-between">
            <div>
              <label class="input-label">{{ t('settlementPools.pool') }}</label>
              <Select v-model="selectedGroupId" :options="groupOptions" searchable />
            </div>
            <button class="btn btn-primary" :disabled="saving || !selectedGroupId" @click="showStartCycleDialog = true">
              <Icon name="play" size="sm" />
              {{ t('settlementPools.startNextCycle') }}
            </button>
          </div>
        </div>

        <SettlementPoolOverview
          v-if="summary"
          :summary="summary"
          :display-estimate="displayEstimate"
        />

        <div class="space-y-6">
          <section v-if="!viewingHistory" class="card p-4">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.parameters') }}</h2>
              <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveConfig">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>

            <div class="space-y-4">
              <div class="grid gap-3 sm:grid-cols-3">
                <div>
                  <label class="input-label">{{ t('settlementPools.totalCostShort') }}</label>
                  <input v-model.number="configForm.total_cost" class="input" type="number" min="0" step="0.01" />
                </div>
                <div>
                  <label class="input-label">{{ t('settlementPools.baseRatio') }}</label>
                  <input v-model.number="configForm.base_ratio" class="input" type="number" min="0" max="1" step="0.01" />
                </div>
                <div>
                  <label class="input-label">{{ t('settlementPools.marketCap') }}</label>
                  <input v-model.number="configForm.market_cap" class="input" type="number" min="0" step="0.01" />
                </div>
              </div>

              <div>
                <div class="mb-2 flex items-center justify-between">
                  <label class="input-label mb-0">{{ t('settlementPools.tiers') }}</label>
                  <button class="btn btn-secondary btn-sm" @click="addTier">{{ t('common.add') }}</button>
                </div>
                <div class="space-y-2">
                  <div v-for="(tier, index) in configForm.tiers" :key="index" class="grid grid-cols-[1fr_1fr_auto] gap-2">
                    <input v-model.number="tier.up_to" class="input" type="number" min="0" step="1" :placeholder="t('settlementPools.unlimited')" />
                    <input v-model.number="tier.weight" class="input" type="number" min="0" step="0.01" />
                    <button class="btn btn-ghost btn-sm" :disabled="configForm.tiers.length <= 1" @click="removeTier(index)">
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section v-if="!viewingHistory" class="card p-4">
            <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.candidates') }}</h2>
              <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
                <UserSearchCombobox
                  class="w-full sm:w-72"
                  :placeholder="t('admin.usage.searchUserPlaceholder')"
                  clear-on-select
                  @select="addCandidate"
                  @search-error="handleUserSearchError"
                />
                <button class="btn btn-primary" :disabled="saving" @click="saveCandidates">{{ t('common.save') }}</button>
              </div>
            </div>

            <DataTable
              :columns="candidateColumns"
              :data="candidateRows"
              row-key="user_id"
              :loading="loading && !!summary"
            >
              <template #cell-user="{ row }">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.email }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  #{{ row.user_id }} <span v-if="row.username">{{ row.username }}</span>
                </div>
              </template>
              <template #cell-actions="{ row }">
                <div class="flex justify-end gap-1">
                  <button
                    class="btn btn-secondary btn-sm"
                    :disabled="saving || actioningUserIds.has(row.user_id)"
                    @click="forceJoinCandidate(row.user_id)"
                  >
                    {{ t('settlementPools.forceJoin') }}
                  </button>
                  <button
                    class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    :title="t('common.delete')"
                    :disabled="saving || actioningUserIds.has(row.user_id)"
                    @click="removeCandidate(row.user_id)"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </template>
              <template #empty>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noCandidates') }}</p>
              </template>
            </DataTable>
          </section>

          <section class="card p-4">
            <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.currentParticipants') }}</h2>
              <div v-if="!viewingHistory" class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
                <UserSearchCombobox
                  class="w-full sm:w-72"
                  :placeholder="t('admin.usage.searchUserPlaceholder')"
                  clear-on-select
                  @select="forceJoinUser"
                  @search-error="handleUserSearchError"
                />
              </div>
            </div>

            <form v-if="!viewingHistory" class="mb-4 grid gap-3 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60 lg:grid-cols-[minmax(200px,1fr)_minmax(220px,1fr)_150px_minmax(220px,1.2fr)_auto] lg:items-end" @submit.prevent="createManualUsageAdjustment">
              <div>
                <label class="input-label">{{ t('settlementPools.manualUsageUser') }}</label>
                <Select
                  v-model="manualUsageForm.user_id"
                  :options="participantOptions"
                  :placeholder="t('settlementPools.manualUsageUserPlaceholder')"
                  searchable
                  :disabled="saving || participantOptions.length === 0"
                />
              </div>
              <div>
                <label class="input-label">{{ t('settlementPools.manualUsageAccount') }}</label>
                <Select
                  v-model="manualUsageForm.account_id"
                  :options="accountOptions"
                  :placeholder="t('settlementPools.manualUsageAccountPlaceholder')"
                  searchable
                  :disabled="saving || accountOptions.length === 0"
                />
              </div>
              <div>
                <label class="input-label">{{ t('settlementPools.manualUsageAmount') }}</label>
                <input v-model.number="manualUsageForm.usage_amount" class="input" type="number" step="0.0001" :disabled="saving" />
              </div>
              <div>
                <label class="input-label">{{ t('settlementPools.manualUsageReason') }}</label>
                <input v-model.trim="manualUsageForm.reason" class="input" type="text" :placeholder="t('settlementPools.manualUsageReasonPlaceholder')" :disabled="saving" />
              </div>
              <button class="btn btn-secondary" type="submit" :disabled="saving || !canSubmitManualUsage">
                <Icon name="plus" size="sm" />
                {{ t('settlementPools.addManualUsage') }}
              </button>
            </form>

            <DataTable
              :columns="participantColumns"
              :data="participantRows"
              row-key="user_id"
              :loading="loading && !!summary"
            >
              <template #cell-user="{ row }">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.email }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  #{{ row.user_id }} <span v-if="row.username">{{ row.username }}</span>
                </div>
              </template>
              <template #cell-raw_usage="{ value }">
                <span class="tabular-nums">{{ usdMoney(value) }}</span>
              </template>
              <template #cell-manual_usage="{ value }">
                <span class="tabular-nums">{{ usdMoney(value) }}</span>
              </template>
              <template #cell-weighted_usage="{ value }">
                <span class="tabular-nums">{{ usdMoney(value) }}</span>
              </template>
              <template #cell-current_tier="{ row }">
                <span class="tabular-nums">{{ currentTierLabel(row.current_tier) }}</span>
              </template>
              <template #cell-fixed_share="{ value }">
                <span class="tabular-nums">{{ cnyMoney(value) }}</span>
              </template>
              <template #cell-dynamic_charge="{ value }">
                <span class="tabular-nums">{{ cnyMoney(value) }}</span>
              </template>
              <template #cell-total_due="{ value }">
                <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ cnyMoney(value) }}</span>
              </template>
              <template #cell-actions="{ row }">
                <div class="flex justify-end">
                  <button
                    class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    :title="t('common.delete')"
                    :disabled="saving || actioningUserIds.has(row.user_id)"
                    @click="removeCurrentParticipant(row.user_id)"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </template>
              <template #empty>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noParticipants') }}</p>
              </template>
            </DataTable>
          </section>

          <section class="card p-4">
            <div class="mb-4">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.accountUsage') }}</h2>
              <p v-if="displayEstimate" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('settlementPools.accountUsagePeriod', { period: settlementCyclePeriod(displayEstimate) }) }}
              </p>
            </div>
            <DataTable
              :columns="accountUsageColumns"
              :data="accountUsageRows"
              row-key="account_id"
              :loading="loading && !!summary"
            >
              <template #cell-account="{ row }">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  #{{ row.account_id }} · {{ row.platform }} / {{ row.type }}
                </div>
              </template>
              <template #cell-total_usage="{ value }">
                <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ usdMoney(value) }}</span>
              </template>
              <template #cell-manual_usage="{ value }">
                <span class="tabular-nums" :class="signedClass(value)">{{ signedUsdMoney(value) }}</span>
              </template>
              <template #cell-weekly_total_usage="{ value }">
                <span class="tabular-nums">{{ usdMoney(value) }}</span>
              </template>
              <template #empty>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noAccountUsage') }}</p>
              </template>
            </DataTable>
          </section>

          <section class="card p-4">
            <div class="mb-4">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.manualUsageAdjustments') }}</h2>
              <p v-if="displayEstimate" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('settlementPools.accountUsagePeriod', { period: settlementCyclePeriod(displayEstimate) }) }}
              </p>
            </div>
            <DataTable
              :columns="manualAdjustmentColumns"
              :data="manualAdjustmentRows"
              row-key="id"
              :loading="loading && !!summary"
            >
              <template #cell-user="{ row }">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.email }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  #{{ row.user_id }} <span v-if="row.username">{{ row.username }}</span>
                </div>
              </template>
              <template #cell-account="{ row }">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.account_name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">#{{ row.account_id }}</div>
              </template>
              <template #cell-usage_amount="{ value }">
                <span class="font-medium tabular-nums" :class="signedClass(value)">{{ signedUsdMoney(value) }}</span>
              </template>
              <template #cell-created_at="{ value }">
                <span class="text-gray-700 dark:text-gray-200">{{ date(value) }}</span>
              </template>
              <template #empty>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noManualUsageAdjustments') }}</p>
              </template>
            </DataTable>
          </section>
        </div>

        <section class="card p-4">
          <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.cycles') }}</h2>
          <DataTable
            :columns="cycleColumns"
            :data="cycleRows"
            row-key="id"
            :loading="loading && !!summary"
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
            <template #cell-total_cost="{ row }">
              <span class="tabular-nums">{{ cnyMoney(row.snapshot?.total_cost ?? row.total_cost) }}</span>
            </template>
            <template #cell-owner_loss="{ row }">
              <span class="tabular-nums">{{ cnyMoney(row.snapshot?.owner_covered_loss ?? 0) }}</span>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex justify-end">
                <button
                  class="btn btn-secondary btn-sm"
                  :disabled="selectedCycleId === row.cycle_id"
                  @click="selectCycle(row.cycle_id)"
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
      </template>
    </div>

    <ConfirmDialog
      :show="showStartCycleDialog"
      :title="t('settlementPools.startNextCycle')"
      :message="t('settlementPools.startConfirm')"
      :confirm-text="t('settlementPools.startNextCycle')"
      @confirm="startNextCycle"
      @cancel="showStartCycleDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import UserSearchCombobox from '@/components/admin/user/UserSearchCombobox.vue'
import SettlementPoolOverview from '@/components/settlement/SettlementPoolOverview.vue'
import { useAppStore } from '@/stores/app'
import * as groupsAPI from '@/api/admin/groups'
import settlementPoolsAPI from '@/api/admin/settlementPools'
import type { SimpleUser } from '@/api/admin/usage'
import type { Column } from '@/components/common/types'
import type { AdminGroup, SettlementPoolAccountUsage, SettlementPoolEstimate, SettlementPoolManualUsageAdjustment, SettlementPoolParticipant, SettlementPoolSummary, SettlementPoolTier } from '@/types'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const groups = ref<AdminGroup[]>([])
const selectedGroupId = ref<number | null>(null)
const summary = ref<SettlementPoolSummary | null>(null)
const selectedCycleId = ref<number | null>(null)
const showStartCycleDialog = ref(false)
const manualCandidates = ref<SettlementPoolParticipant[]>([])
const removedCandidateIds = ref<Set<number>>(new Set())
const actioningUserIds = ref<Set<number>>(new Set())
const manualUsageForm = reactive({
  user_id: null as number | null,
  account_id: null as number | null,
  usage_amount: null as number | null,
  reason: ''
})

const configForm = reactive({
  total_cost: 0,
  base_ratio: 0.2,
  market_cap: 0.35,
  tiers: [
    { up_to: 300, weight: 1 },
    { up_to: 1200, weight: 0.8 },
    { up_to: null, weight: 0.64 }
  ] as SettlementPoolTier[]
})

const settlementGroups = computed(() => groups.value.filter(group => group.subscription_type === 'settlement_pool'))
const groupOptions = computed(() => settlementGroups.value.map(group => ({ value: group.id, label: `${group.name} #${group.id}` })))
const rightAlignedColumnClass = 'text-right [&>div]:justify-end'
const viewingHistory = computed(() => selectedCycleId.value != null)
const displayEstimate = computed<SettlementPoolEstimate | null>(() => {
  if (!summary.value) return null
  if (selectedCycleId.value != null) {
    return summary.value.cycles.find(cycle => cycle.id === selectedCycleId.value)?.snapshot ?? null
  }
  return summary.value.estimate ?? null
})
const participantRows = computed(() => {
  return displayEstimate.value?.participants || []
})
const participantOptions = computed(() => participantRows.value.map(row => ({
  value: row.user_id,
  label: `${row.email || row.username || `#${row.user_id}`} #${row.user_id}`
})))
const canSubmitManualUsage = computed(() => {
  return Number(manualUsageForm.user_id || 0) > 0 &&
    Number(manualUsageForm.account_id || 0) > 0 &&
    Number(manualUsageForm.usage_amount || 0) !== 0 &&
    manualUsageForm.reason.trim().length > 0
})
const accountUsageRows = computed<SettlementPoolAccountUsage[]>(() => {
  return displayEstimate.value?.account_usage || []
})
const accountOptions = computed(() => accountUsageRows.value.map(row => ({
  value: row.account_id,
  label: `${row.name || `#${row.account_id}`} #${row.account_id}`
})))
const manualAdjustmentRows = computed<SettlementPoolManualUsageAdjustment[]>(() => {
  return displayEstimate.value?.manual_adjustments || []
})
const candidateRows = computed<SettlementPoolParticipant[]>(() => {
  const byID = new Map<number, SettlementPoolParticipant>()
  for (const row of summary.value?.candidates || []) {
    if (!removedCandidateIds.value.has(row.user_id)) byID.set(row.user_id, row)
  }
  for (const row of manualCandidates.value) byID.set(row.user_id, row)
  return [...byID.values()].sort((a, b) => a.user_id - b.user_id)
})
const candidateColumns = computed<Column[]>(() => [
  { key: 'user', label: t('settlementPools.user'), class: 'min-w-[220px]' },
  { key: 'actions', label: '', class: rightAlignedColumnClass }
])
const participantColumns = computed<Column[]>(() => [
  { key: 'user', label: t('settlementPools.user'), class: 'min-w-[220px]' },
  { key: 'raw_usage', label: t('settlementPools.rawUsage'), class: rightAlignedColumnClass },
  { key: 'manual_usage', label: t('settlementPools.manualUsage'), class: rightAlignedColumnClass },
  { key: 'weighted_usage', label: t('settlementPools.weightedUsage'), class: rightAlignedColumnClass },
  { key: 'current_tier', label: t('settlementPools.currentTier'), class: rightAlignedColumnClass },
  { key: 'fixed_share', label: t('settlementPools.fixedShare'), class: rightAlignedColumnClass },
  { key: 'dynamic_charge', label: t('settlementPools.dynamicCharge'), class: rightAlignedColumnClass },
  { key: 'total_due', label: t('settlementPools.totalDue'), class: rightAlignedColumnClass },
  ...(viewingHistory.value ? [] : [{ key: 'actions', label: '', class: rightAlignedColumnClass }])
])
const cycleColumns = computed<Column[]>(() => [
  { key: 'period', label: t('settlementPools.period'), class: 'min-w-[260px]' },
  { key: 'status', label: t('common.status') },
  { key: 'total_cost', label: t('settlementPools.totalCost'), class: rightAlignedColumnClass },
  { key: 'owner_loss', label: t('settlementPools.ownerLoss'), class: rightAlignedColumnClass },
  { key: 'actions', label: '', class: rightAlignedColumnClass }
])
const accountUsageColumns = computed<Column[]>(() => [
  { key: 'account', label: t('settlementPools.account'), class: 'min-w-[220px]' },
  { key: 'total_usage', label: t('settlementPools.cycleUsage'), class: rightAlignedColumnClass },
  { key: 'manual_usage', label: t('settlementPools.manualUsage'), class: rightAlignedColumnClass },
  { key: 'weekly_total_usage', label: t('settlementPools.weeklyUsage'), class: rightAlignedColumnClass }
])
const manualAdjustmentColumns = computed<Column[]>(() => [
  { key: 'user', label: t('settlementPools.user'), class: 'min-w-[220px]' },
  { key: 'account', label: t('settlementPools.account'), class: 'min-w-[200px]' },
  { key: 'usage_amount', label: t('settlementPools.manualUsageAmount'), class: rightAlignedColumnClass },
  { key: 'reason', label: t('settlementPools.manualUsageReason'), class: 'min-w-[240px]' },
  { key: 'created_at', label: t('settlementPools.createdAt'), class: 'min-w-[180px]' }
])

type CycleRow = {
  id: string
  cycle_id: number | null
  group_id: number
  status: 'active' | 'locked'
  started_at: string
  ended_at?: string | null
  total_cost: number
  base_ratio: number
  market_cap: number
  tiers: SettlementPoolTier[]
  snapshot?: SettlementPoolEstimate | null
  created_at: string
  updated_at: string
}

const cycleRows = computed<CycleRow[]>(() => {
  if (!summary.value) return []
  const rows: CycleRow[] = []
  if (summary.value.estimate) {
    rows.push({
      ...(summary.value.active_cycle ?? {
        group_id: summary.value.group.id,
        status: summary.value.estimate.status,
        started_at: summary.value.estimate.started_at,
        ended_at: summary.value.estimate.ended_at,
        total_cost: summary.value.estimate.total_cost,
        base_ratio: summary.value.estimate.base_ratio,
        market_cap: summary.value.estimate.market_cap,
        tiers: summary.value.estimate.tiers,
        created_at: summary.value.estimate.started_at,
        updated_at: summary.value.estimate.started_at
      }),
      id: `active-${summary.value.group.id}`,
      cycle_id: null,
      snapshot: summary.value.estimate
    } as CycleRow)
  }
  const activeCycleID = summary.value.active_cycle?.id
  return rows.concat((summary.value.cycles || [])
    .filter(cycle => cycle.status !== 'active' && cycle.id !== activeCycleID)
    .map(cycle => ({
      ...cycle,
      id: String(cycle.id),
      cycle_id: cycle.id,
      snapshot: cycle.snapshot ?? null
    })))
})

watch(selectedGroupId, () => {
  selectedCycleId.value = null
  if (selectedGroupId.value) loadSummary()
})

function syncForm(next: SettlementPoolSummary | null) {
  const cycle = next?.active_cycle
  const config = next?.config
  configForm.total_cost = roundedInputNumber(cycle?.total_cost ?? 0)
  configForm.base_ratio = roundedInputNumber(cycle?.base_ratio ?? config?.base_ratio ?? 0.2)
  configForm.market_cap = roundedInputNumber(cycle?.market_cap ?? config?.market_cap ?? 0.35)
  configForm.tiers = (cycle?.tiers ?? config?.tiers ?? configForm.tiers).map(tier => ({
    up_to: tier.up_to == null ? null : roundedInputNumber(tier.up_to),
    weight: roundedInputNumber(tier.weight)
  }))
  selectedCycleId.value = null
  manualCandidates.value = []
  removedCandidateIds.value = new Set()
  actioningUserIds.value = new Set()
  resetManualUsageForm()
}

function roundedInputNumber(value: number) {
  return Number(Number(value || 0).toFixed(6))
}

async function loadGroups() {
  groups.value = await groupsAPI.getAll()
  if (!selectedGroupId.value && settlementGroups.value.length > 0) {
    const requestedGroup = Number(firstQueryValue(route.query.group) ?? firstQueryValue(route.query.groupId))
    const matchedGroup = settlementGroups.value.find(group => group.id === requestedGroup)
    selectedGroupId.value = matchedGroup?.id ?? settlementGroups.value[0].id
  }
}

async function loadSummary() {
  if (!selectedGroupId.value) return
  loading.value = true
  try {
    summary.value = await settlementPoolsAPI.getSummary(selectedGroupId.value)
    syncForm(summary.value)
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function refresh() {
  loading.value = true
  try {
    await loadGroups()
    if (selectedGroupId.value) await loadSummary()
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  if (!selectedGroupId.value) return
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.updateConfig(selectedGroupId.value, {
      total_cost: finiteNumber(configForm.total_cost),
      base_ratio: finiteNumber(configForm.base_ratio),
      market_cap: finiteNumber(configForm.market_cap),
      tiers: configForm.tiers.map(tier => ({
        up_to: optionalFiniteNumber(tier.up_to),
        weight: finiteNumber(tier.weight)
      }))
    })
    syncForm(summary.value)
    appStore.showSuccess(t('settlementPools.saved'))
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToSave'))
  } finally {
    saving.value = false
  }
}

async function saveCandidates() {
  if (!selectedGroupId.value) return
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.syncCandidates(
      selectedGroupId.value,
      candidateRows.value.map(row => row.user_id)
    )
    syncForm(summary.value)
    appStore.showSuccess(t('settlementPools.saved'))
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToSave'))
  } finally {
    saving.value = false
  }
}

async function startNextCycle() {
  if (!selectedGroupId.value) return
  showStartCycleDialog.value = false
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.startNextCycle(selectedGroupId.value)
    syncForm(summary.value)
    appStore.showSuccess(t('settlementPools.cycleStarted'))
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToSave'))
  } finally {
    saving.value = false
  }
}

function handleUserSearchError() {
  appStore.showError(t('settlementPools.failedToSearchUsers'))
}

function addCandidate(user: SimpleUser) {
  if (candidateRows.value.some(row => row.user_id === user.id)) return
  const removed = new Set(removedCandidateIds.value)
  removed.delete(user.id)
  removedCandidateIds.value = removed
  if (summary.value?.candidates?.some(row => row.user_id === user.id)) return
  manualCandidates.value.push({
    user_id: user.id,
    email: user.email,
    username: '',
    status: 'active',
    created_at: new Date().toISOString()
  })
}

function removeCandidate(userId: number) {
  manualCandidates.value = manualCandidates.value.filter(row => row.user_id !== userId)
  const removed = new Set(removedCandidateIds.value)
  removed.add(userId)
  removedCandidateIds.value = removed
}

async function forceJoinCandidate(userId: number) {
  await forceJoinUserId(userId)
}

async function forceJoinUser(user: SimpleUser) {
  await forceJoinUserId(user.id)
}

async function forceJoinUserId(userId: number) {
  if (!selectedGroupId.value) return
  actioningUserIds.value = new Set([...actioningUserIds.value, userId])
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.forceJoinCurrentCycle(selectedGroupId.value, [userId])
    syncForm(summary.value)
    appStore.showSuccess(t('settlementPools.joinedCurrentCycle'))
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToSave'))
  } finally {
    const next = new Set(actioningUserIds.value)
    next.delete(userId)
    actioningUserIds.value = next
    saving.value = false
  }
}

async function removeCurrentParticipant(userId: number) {
  if (!selectedGroupId.value) return
  actioningUserIds.value = new Set([...actioningUserIds.value, userId])
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.removeCurrentParticipant(selectedGroupId.value, userId)
    syncForm(summary.value)
    appStore.showSuccess(t('settlementPools.saved'))
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToSave'))
  } finally {
    const next = new Set(actioningUserIds.value)
    next.delete(userId)
    actioningUserIds.value = next
    saving.value = false
  }
}

async function createManualUsageAdjustment() {
  if (!selectedGroupId.value || !canSubmitManualUsage.value) return
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.createManualUsageAdjustment(selectedGroupId.value, {
      user_id: Number(manualUsageForm.user_id),
      account_id: Number(manualUsageForm.account_id),
      usage_amount: finiteNumber(manualUsageForm.usage_amount),
      reason: manualUsageForm.reason.trim()
    })
    syncForm(summary.value)
    appStore.showSuccess(t('settlementPools.manualUsageAdded'))
  } catch (error: any) {
    appStore.showError(error?.message || t('settlementPools.failedToAddManualUsage'))
  } finally {
    saving.value = false
  }
}

function resetManualUsageForm() {
  manualUsageForm.user_id = null
  manualUsageForm.account_id = null
  manualUsageForm.usage_amount = null
  manualUsageForm.reason = ''
}

function addTier() {
  configForm.tiers.push({ up_to: null, weight: 1 })
}

function removeTier(index: number) {
  configForm.tiers.splice(index, 1)
}

function selectCycle(cycleId: number | null) {
  selectedCycleId.value = cycleId
}

function usdMoney(value: number | null | undefined) {
  return `$${Number(value || 0).toFixed(4)}`
}

function signedUsdMoney(value: number | null | undefined) {
  const numberValue = Number(value || 0)
  const sign = numberValue > 0 ? '+' : ''
  return `${sign}$${numberValue.toFixed(4)}`
}

function signedClass(value: number | null | undefined) {
  const numberValue = Number(value || 0)
  if (numberValue > 0) return 'text-green-600 dark:text-green-400'
  if (numberValue < 0) return 'text-red-600 dark:text-red-400'
  return 'text-gray-700 dark:text-gray-200'
}

function cnyMoney(value: number | null | undefined) {
  return `¥${Number(value || 0).toFixed(4)}`
}

function date(value?: string | null) {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

function settlementCyclePeriod(estimate: SettlementPoolEstimate): string {
  return `${date(estimate.started_at)} - ${estimate.ended_at ? date(estimate.ended_at) : t('settlementPools.status.active')}`
}

function tierLabel(tiers: SettlementPoolTier[], index: number) {
  const previous = index === 0 ? 0 : tiers[index - 1]?.up_to
  const current = tiers[index]?.up_to
  if (current == null) return `${previous ?? 0}+`
  return `${previous ?? 0}-${current}`
}

function currentTierLabel(index: number) {
  const tiers = displayEstimate.value?.tiers || []
  const tier = tiers[index]
  if (!tier) return ''
  return `${tierLabel(tiers, index)} x ${tier.weight}`
}

function firstQueryValue(value: unknown) {
  if (Array.isArray(value)) return value[0]
  return value
}

function optionalFiniteNumber(value: unknown) {
  if (value === null || value === undefined || value === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

function finiteNumber(value: unknown) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

onMounted(refresh)
</script>
