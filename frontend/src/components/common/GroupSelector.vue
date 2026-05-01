<template>
  <div>
    <label class="input-label">
      {{ t('admin.users.groups') }}
      <span class="font-normal text-gray-400">{{ t('common.selectedCount', { count: modelValue.length }) }}</span>
    </label>
    <div
      class="grid max-h-32 grid-cols-2 gap-1 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-800"
    >
      <label
        v-for="group in filteredGroups"
        :key="group.id"
        class="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 transition-colors hover:bg-white dark:hover:bg-dark-700"
        :title="groupTitle(group)"
      >
        <input
          type="checkbox"
          :value="group.id"
          :checked="modelValue.includes(group.id)"
          @change="handleChange(group.id, ($event.target as HTMLInputElement).checked)"
          class="h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
        />
        <GroupBadge
          :name="group.name"
          :platform="group.platform"
          :subscription-type="group.subscription_type"
          :rate-multiplier="group.rate_multiplier"
          class="min-w-0 flex-1"
        />
        <span v-if="showAccountCount" class="shrink-0 text-xs text-gray-400">{{ accountCount(group) }}</span>
      </label>
      <div
        v-if="filteredGroups.length === 0"
        class="col-span-2 py-2 text-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('common.noGroupsAvailable') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from './GroupBadge.vue'
import type { Group, GroupPlatform } from '@/types'

const { t } = useI18n()

interface Props {
  modelValue: number[]
  groups: Group[]
  platform?: GroupPlatform // Optional platform filter
  mixedScheduling?: boolean // For antigravity accounts: allow anthropic/gemini groups
  maxSelection?: number
  showAccountCount?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showAccountCount: true,
})
const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

// Filter groups by platform if specified
const filteredGroups = computed(() => {
  if (!props.platform) {
    return props.groups
  }
  // antigravity 账户启用混合调度后，可选择 anthropic/gemini 分组
  if (props.platform === 'antigravity' && props.mixedScheduling) {
    return props.groups.filter(
      (g) => g.platform === 'antigravity' || g.platform === 'anthropic' || g.platform === 'gemini'
    )
  }
  // 默认：只能选择同 platform 的分组
  return props.groups.filter((g) => g.platform === props.platform)
})

const handleChange = (groupId: number, checked: boolean) => {
  if (checked && props.maxSelection === 1) {
    emit('update:modelValue', [groupId])
    return
  }
  const newValue = checked
    ? [...props.modelValue, groupId]
    : props.modelValue.filter((id) => id !== groupId)
  emit('update:modelValue', newValue)
}

const accountCount = (group: Group) => {
  return 'account_count' in group && typeof group.account_count === 'number'
    ? group.account_count
    : 0
}

const groupTitle = (group: Group) => {
  if (!props.showAccountCount) {
    return group.name
  }
  return t('admin.groups.rateAndAccounts', { rate: group.rate_multiplier, count: accountCount(group) })
}
</script>
