<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="summary">
        <!-- Balance Summary -->
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('resourceSupply.stats.available') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(summary.balance.available_amount) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('resourceSupply.stats.lifetimeEarned') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(summary.balance.lifetime_earned_amount) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('resourceSupply.stats.lifetimeTransferred') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(summary.balance.lifetime_transferred_amount) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('resourceSupply.stats.accounts') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ summary.accounts.length }}
            </p>
          </div>
        </div>

        <!-- Transfer -->
        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('resourceSupply.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('resourceSupply.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || summary.balance.available_amount <= 0"
              @click="doTransfer"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('resourceSupply.transfer.transferring') : t('resourceSupply.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="summary.balance.available_amount <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('resourceSupply.transfer.empty') }}
          </p>
        </div>

        <!-- Submit Account (Self-service) -->
        <div v-if="summary.self_service_enabled" class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('resourceSupply.submit.title') }}</h3>
              <p v-if="availableSupplyGroups.length === 0" class="mt-1 text-sm text-amber-600 dark:text-amber-400">
                {{ t('resourceSupply.submit.noGroups') }}
              </p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="availableSupplyGroups.length === 0"
              @click="showCreateModal = true"
            >
              <Icon name="plus" size="sm" />
              <span>{{ t('resourceSupply.submit.title') }}</span>
            </button>
          </div>
        </div>

        <CreateAccountModal
          :show="showCreateModal"
          mode="resource-supply"
          :supply-groups="availableSupplyGroups"
          @close="showCreateModal = false"
          @created="onAccountCreated"
        />

        <!-- Owned Accounts Table -->
        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white mb-4">{{ t('resourceSupply.accounts.title') }}</h3>
          <div v-if="summary.accounts.length === 0" class="rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('resourceSupply.accounts.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full min-w-[700px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.accounts.columns.name') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.accounts.columns.platform') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.accounts.columns.groups') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.accounts.columns.status') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.accounts.columns.source') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.accounts.columns.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="acct in summary.accounts" :key="acct.id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ acct.name }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ acct.platform }} / {{ acct.type }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ (acct.group_names || []).join(', ') || '-' }}</td>
                  <td class="px-3 py-3">
                    <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" :class="statusBadgeClass(acct.supply_status)">
                      {{ acct.supply_status }}
                    </span>
                    <p v-if="acct.supply_status_reason" class="mt-0.5 text-xs text-gray-500 dark:text-gray-400 truncate max-w-[200px]" :title="acct.supply_status_reason">
                      {{ acct.supply_status_reason }}
                    </p>
                  </td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ acct.supply_source }}</td>
                  <td class="px-3 py-3">
                    <div v-if="acct.supply_source === 'self_service' && (acct.supply_status === 'schedulable' || acct.supply_status === 'paused')" class="flex gap-1">
                      <button v-if="acct.supply_status === 'schedulable'" class="btn btn-sm btn-secondary" @click="pauseAccount(acct.id)">
                        {{ t('resourceSupply.accounts.pause') }}
                      </button>
                      <button class="btn btn-sm text-red-600 hover:text-red-700" @click="revokeAccount(acct.id)">
                        {{ t('resourceSupply.accounts.revoke') }}
                      </button>
                    </div>
                    <span v-else class="text-gray-400">-</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Ledger Table -->
        <div class="card p-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('resourceSupply.ledger.title') }}</h3>
            <select v-model="ledgerTypeFilter" class="input w-auto min-w-[120px]" @change="loadLedger(1)">
              <option value="">{{ t('resourceSupply.ledger.allTypes') }}</option>
              <option value="reward">{{ t('resourceSupply.ledger.reward') }}</option>
              <option value="transfer">{{ t('resourceSupply.ledger.transfer') }}</option>
              <option value="adjustment">{{ t('resourceSupply.ledger.adjustment') }}</option>
            </select>
          </div>

          <div v-if="ledgerLoading" class="flex justify-center py-6">
            <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
          </div>
          <div v-else-if="ledgerEntries.length === 0" class="rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('resourceSupply.ledger.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full min-w-[800px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.ledger.columns.type') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('resourceSupply.ledger.columns.amount') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('resourceSupply.ledger.columns.balanceAfter') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.ledger.columns.group') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.ledger.columns.model') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.ledger.columns.note') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('resourceSupply.ledger.columns.time') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in ledgerEntries" :key="entry.id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3">
                    <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" :class="ledgerTypeBadgeClass(entry.ledger_type)">
                      {{ entry.ledger_type }}
                    </span>
                  </td>
                  <td class="px-3 py-3 text-right font-medium" :class="entry.amount >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'">
                    {{ entry.amount >= 0 ? '+' : '' }}{{ formatCurrency(entry.amount) }}
                  </td>
                  <td class="px-3 py-3 text-right text-gray-700 dark:text-gray-300">{{ formatCurrency(entry.balance_after) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ entry.group_name || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ entry.model || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300 truncate max-w-[150px]" :title="entry.note || undefined">{{ entry.note || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(entry.created_at) }}</td>
                </tr>
              </tbody>
            </table>

            <!-- Pagination -->
            <div v-if="ledgerTotal > ledgerPageSize" class="mt-4 flex items-center justify-between text-sm text-gray-500">
              <span>{{ t('resourceSupply.ledger.total', { count: ledgerTotal }) }}</span>
              <div class="flex gap-1">
                <button class="btn btn-sm btn-secondary" :disabled="ledgerPage <= 1" @click="loadLedger(ledgerPage - 1)">{{ t('common.previous') }}</button>
                <button class="btn btn-sm btn-secondary" :disabled="ledgerPage >= ledgerPages" @click="loadLedger(ledgerPage + 1)">{{ t('common.next') }}</button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import CreateAccountModal from '@/components/account/CreateAccountModal.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import resourceSupplyAPI from '@/api/resourceSupply'
import userGroupsAPI from '@/api/groups'
import type { Group, ResourceSupplySummary, ResourceSupplyLedgerEntry } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(true)
const transferring = ref(false)
const summary = ref<ResourceSupplySummary | null>(null)
const availableGroups = ref<Group[]>([])
const availableSupplyGroups = computed(() =>
  availableGroups.value.filter((group) =>
    group.platform === 'openai' &&
    group.supply_rewards_enabled &&
    group.status === 'active',
  ),
)

// Create account modal state
const showCreateModal = ref(false)

// Ledger state
const ledgerLoading = ref(false)
const ledgerEntries = ref<ResourceSupplyLedgerEntry[]>([])
const ledgerTypeFilter = ref('')
const ledgerPage = ref(1)
const ledgerPageSize = ref(20)
const ledgerTotal = ref(0)
const ledgerPages = ref(0)

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'schedulable': return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'pending_review': return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400'
    case 'testing': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
    case 'paused': return 'bg-gray-100 text-gray-800 dark:bg-gray-700/30 dark:text-gray-400'
    case 'rejected': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'revoked': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    default: return 'bg-gray-100 text-gray-600 dark:bg-gray-700/30 dark:text-gray-400'
  }
}

function ledgerTypeBadgeClass(type: string): string {
  switch (type) {
    case 'reward': return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'transfer': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
    case 'adjustment': return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400'
    default: return 'bg-gray-100 text-gray-600 dark:bg-gray-700/30 dark:text-gray-400'
  }
}

async function loadSummary(silent = false): Promise<void> {
  if (!silent) loading.value = true
  try {
    summary.value = await resourceSupplyAPI.getSummary()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('resourceSupply.loadFailed')))
  } finally {
    if (!silent) loading.value = false
  }
}

async function loadLedger(page = 1): Promise<void> {
  ledgerLoading.value = true
  ledgerPage.value = page
  try {
    const resp = await resourceSupplyAPI.getLedger({
      type: ledgerTypeFilter.value || undefined,
      page,
      page_size: ledgerPageSize.value,
    })
    ledgerEntries.value = resp.items
    ledgerTotal.value = resp.total
    ledgerPages.value = resp.pages
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('resourceSupply.loadFailed')))
  } finally {
    ledgerLoading.value = false
  }
}

async function loadAvailableGroups(): Promise<void> {
  try {
    availableGroups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('resourceSupply.submit.groupsLoadFailed')))
  }
}

async function doTransfer(): Promise<void> {
  if (!summary.value || summary.value.balance.available_amount <= 0 || transferring.value) return
  transferring.value = true
  try {
    const result = await resourceSupplyAPI.transfer()
    appStore.showSuccess(t('resourceSupply.transfer.success', { amount: formatCurrency(result.amount) }))
    await Promise.all([
      loadSummary(true),
      loadLedger(1),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('resourceSupply.transfer.failed')))
  } finally {
    transferring.value = false
  }
}

async function onAccountCreated(): Promise<void> {
  showCreateModal.value = false
  await loadSummary(true)
}

async function pauseAccount(id: number): Promise<void> {
  try {
    await resourceSupplyAPI.pauseAccount(id, '')
    appStore.showSuccess(t('resourceSupply.accounts.pauseSuccess'))
    await loadSummary(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('resourceSupply.accounts.actionFailed')))
  }
}

async function revokeAccount(id: number): Promise<void> {
  try {
    await resourceSupplyAPI.revokeAccount(id, '')
    appStore.showSuccess(t('resourceSupply.accounts.revokeSuccess'))
    await loadSummary(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('resourceSupply.accounts.actionFailed')))
  }
}

onMounted(() => {
  void loadSummary()
  void loadAvailableGroups()
  void loadLedger()
})
</script>
