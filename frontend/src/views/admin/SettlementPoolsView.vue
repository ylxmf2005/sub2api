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

        <SettlementPoolOverview v-if="summary" :summary="summary" />

        <div class="grid gap-6 xl:grid-cols-[420px_1fr]">
          <section class="card p-4">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.parameters') }}</h2>
              <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveConfig">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>

            <div class="space-y-4">
              <div class="grid grid-cols-3 gap-3">
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

          <section class="card p-4">
            <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.participants') }}</h2>
              <div class="flex gap-2">
                <input v-model.trim="userSearch" class="input w-64" :placeholder="t('settlementPools.searchUser')" @keyup.enter="searchUsers" />
                <button class="btn btn-secondary" @click="searchUsers">{{ t('common.search') }}</button>
                <button class="btn btn-primary" :disabled="saving" @click="saveParticipants">{{ t('common.save') }}</button>
              </div>
            </div>

            <div v-if="searchResults.length > 0" class="mb-4 flex flex-wrap gap-2">
              <button
                v-for="user in searchResults"
                :key="user.id"
                class="rounded-md border border-gray-200 px-3 py-1.5 text-left text-xs hover:border-primary-300 hover:bg-primary-50 dark:border-dark-600 dark:hover:bg-primary-900/20"
                @click="addParticipant(user)"
              >
                <span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
                <span class="ml-2 text-gray-500">#{{ user.id }}</span>
              </button>
            </div>

            <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
              <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
                <thead class="bg-gray-50 dark:bg-dark-700">
                  <tr>
                    <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.user') }}</th>
                    <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.rawUsage') }}</th>
                    <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.weightedUsage') }}</th>
                    <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.currentTier') }}</th>
                    <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.fixedShare') }}</th>
                    <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.totalDue') }}</th>
                    <th class="px-3 py-2"></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
                  <tr v-for="row in participantRows" :key="row.user_id">
                    <td class="px-3 py-2">
                      <div class="font-medium text-gray-900 dark:text-white">{{ row.email }}</div>
                      <div class="text-xs text-gray-500">#{{ row.user_id }} {{ row.username }}</div>
                    </td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ money(row.raw_usage) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ money(row.weighted_usage) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ currentTierLabel(row.current_tier) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ money(row.fixed_share) }}</td>
                    <td class="px-3 py-2 text-right font-medium tabular-nums text-gray-900 dark:text-white">{{ money(row.total_due) }}</td>
                    <td class="px-3 py-2 text-right">
                      <button class="btn btn-ghost btn-sm" @click="removeParticipant(row.user_id)">
                        <Icon name="trash" size="sm" />
                      </button>
                    </td>
                  </tr>
                  <tr v-if="participantRows.length === 0">
                    <td colspan="7" class="px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('settlementPools.noParticipants') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </div>

        <section class="card p-4">
          <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('settlementPools.cycles') }}</h2>
          <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
              <thead class="bg-gray-50 dark:bg-dark-700">
                <tr>
                  <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.period') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('common.status') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.totalCost') }}</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('settlementPools.ownerLoss') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
                <tr v-for="cycle in summary?.cycles || []" :key="cycle.id">
                  <td class="px-3 py-2 text-gray-700 dark:text-gray-200">{{ date(cycle.started_at) }} - {{ cycle.ended_at ? date(cycle.ended_at) : t('settlementPools.status.active') }}</td>
                  <td class="px-3 py-2">
                    <span class="badge" :class="cycle.status === 'active' ? 'badge-success' : 'badge-secondary'">{{ t(`settlementPools.status.${cycle.status}`) }}</span>
                  </td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ money(cycle.snapshot?.total_cost ?? cycle.total_cost) }}</td>
                  <td class="px-3 py-2 text-right tabular-nums">{{ money(cycle.snapshot?.owner_covered_loss ?? 0) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
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
import Icon from '@/components/icons/Icon.vue'
import SettlementPoolOverview from '@/components/settlement/SettlementPoolOverview.vue'
import { useAppStore } from '@/stores/app'
import * as groupsAPI from '@/api/admin/groups'
import * as usersAPI from '@/api/admin/users'
import settlementPoolsAPI from '@/api/admin/settlementPools'
import type { AdminGroup, AdminUser, SettlementPoolParticipantEstimate, SettlementPoolSummary, SettlementPoolTier } from '@/types'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const groups = ref<AdminGroup[]>([])
const selectedGroupId = ref<number | null>(null)
const summary = ref<SettlementPoolSummary | null>(null)
const showStartCycleDialog = ref(false)
const userSearch = ref('')
const searchResults = ref<AdminUser[]>([])
const manualParticipants = ref<SettlementPoolParticipantEstimate[]>([])
const removedParticipantIds = ref<Set<number>>(new Set())

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
const participantRows = computed(() => {
  const fromEstimate = summary.value?.estimate?.participants || []
  const byID = new Map<number, SettlementPoolParticipantEstimate>()
  for (const row of fromEstimate) {
    if (!removedParticipantIds.value.has(row.user_id)) byID.set(row.user_id, row)
  }
  for (const row of manualParticipants.value) byID.set(row.user_id, row)
  return [...byID.values()].sort((a, b) => a.user_id - b.user_id)
})

watch(selectedGroupId, () => {
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
  manualParticipants.value = []
  removedParticipantIds.value = new Set()
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

async function saveParticipants() {
  if (!selectedGroupId.value) return
  saving.value = true
  try {
    summary.value = await settlementPoolsAPI.syncParticipants(
      selectedGroupId.value,
      participantRows.value.map(row => row.user_id)
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

async function searchUsers() {
  if (!userSearch.value) return
  const res = await usersAPI.list(1, 8, { search: userSearch.value, status: 'active' })
  searchResults.value = res.items
}

function addParticipant(user: AdminUser) {
  if (participantRows.value.some(row => row.user_id === user.id)) return
  const removed = new Set(removedParticipantIds.value)
  removed.delete(user.id)
  removedParticipantIds.value = removed
  if (summary.value?.estimate?.participants.some(row => row.user_id === user.id)) return
  manualParticipants.value.push({
    user_id: user.id,
    email: user.email,
    username: user.username,
    status: user.status,
    raw_usage: 0,
    weighted_usage: 0,
    current_tier: 0,
    fixed_share: 0,
    dynamic_charge: 0,
    total_due: 0
  })
}

function removeParticipant(userId: number) {
  manualParticipants.value = manualParticipants.value.filter(row => row.user_id !== userId)
  const removed = new Set(removedParticipantIds.value)
  removed.add(userId)
  removedParticipantIds.value = removed
}

function addTier() {
  configForm.tiers.push({ up_to: null, weight: 1 })
}

function removeTier(index: number) {
  configForm.tiers.splice(index, 1)
}

function money(value: number | null | undefined) {
  return `$${Number(value || 0).toFixed(4)}`
}

function date(value: string) {
  return new Date(value).toLocaleString()
}

function tierLabel(tiers: SettlementPoolTier[], index: number) {
  const previous = index === 0 ? 0 : tiers[index - 1]?.up_to
  const current = tiers[index]?.up_to
  if (current == null) return `${previous ?? 0}+`
  return `${previous ?? 0}-${current}`
}

function currentTierLabel(index: number) {
  const tiers = summary.value?.estimate?.tiers || []
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
