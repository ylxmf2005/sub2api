<template>
  <div ref="rootRef" class="relative">
    <div class="relative">
      <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
        <Icon name="search" size="sm" class="text-gray-400" />
      </div>
      <input
        v-model="keyword"
        type="text"
        autocomplete="off"
        class="input w-full pl-9 pr-9"
        :placeholder="placeholder"
        @focus="openDropdown"
        @input="handleInput"
        @keydown.escape="closeDropdown"
        @keydown.down.prevent="moveActive(1)"
        @keydown.up.prevent="moveActive(-1)"
        @keydown.enter.prevent="selectActive"
      />
      <button
        v-if="keyword"
        type="button"
        class="absolute inset-y-0 right-0 flex items-center px-3 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-gray-200"
        :title="t('common.clear')"
        @click="clear"
      >
        <Icon name="x" size="sm" />
      </button>
    </div>

    <div
      v-if="dropdownOpen && (loading || results.length > 0 || searched)"
      class="absolute left-0 right-0 top-full z-30 mt-1 max-h-64 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
    >
      <div v-if="loading" class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>
      <button
        v-for="(user, index) in results"
        :key="user.id"
        type="button"
        class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors"
        :class="index === activeIndex ? 'bg-primary-50 dark:bg-primary-900/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700'"
        @mousedown.prevent="selectUser(user)"
      >
        <span class="min-w-0">
          <span class="block truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            #{{ user.id }}<span v-if="user.username"> - {{ user.username }}</span>
          </span>
        </span>
      </button>
      <div v-if="!loading && results.length === 0 && searched" class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.noData') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { searchUsers, type SimpleUser } from '@/api/admin/usage'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  placeholder?: string
  clearOnSelect?: boolean
  debounceMs?: number
}>(), {
  placeholder: '',
  clearOnSelect: false,
  debounceMs: 300
})

const emit = defineEmits<{
  (e: 'select', user: SimpleUser): void
  (e: 'search-error', error: unknown): void
}>()

const { t } = useI18n()
const rootRef = ref<HTMLElement | null>(null)
const keyword = ref('')
const results = ref<SimpleUser[]>([])
const loading = ref(false)
const searched = ref(false)
const dropdownOpen = ref(false)
const activeIndex = ref(-1)

let searchTimer: number | null = null
let requestVersion = 0

function openDropdown() {
  dropdownOpen.value = true
}

function closeDropdown() {
  dropdownOpen.value = false
  activeIndex.value = -1
}

function handleInput() {
  openDropdown()
  searched.value = false
  activeIndex.value = -1
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(runSearch, props.debounceMs)
}

async function runSearch() {
  const query = keyword.value.trim()
  const version = ++requestVersion
  if (!query) {
    loading.value = false
    searched.value = false
    results.value = []
    return
  }

  loading.value = true
  try {
    const users = await searchUsers(query)
    if (version !== requestVersion) return
    results.value = users
    searched.value = true
    activeIndex.value = users.length > 0 ? 0 : -1
  } catch (error) {
    if (version !== requestVersion) return
    results.value = []
    searched.value = true
    emit('search-error', error)
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

function moveActive(delta: number) {
  if (!dropdownOpen.value) openDropdown()
  if (results.value.length === 0) return
  activeIndex.value = (activeIndex.value + delta + results.value.length) % results.value.length
}

function selectActive() {
  if (activeIndex.value < 0) return
  const user = results.value[activeIndex.value]
  if (user) selectUser(user)
}

function selectUser(user: SimpleUser) {
  emit('select', user)
  if (props.clearOnSelect) {
    keyword.value = ''
    results.value = []
    searched.value = false
  } else {
    keyword.value = user.email
  }
  closeDropdown()
}

function clear() {
  requestVersion++
  keyword.value = ''
  results.value = []
  searched.value = false
  closeDropdown()
}

function handleDocumentClick(event: MouseEvent) {
  const target = event.target as Node
  if (!rootRef.value?.contains(target)) closeDropdown()
}

document.addEventListener('mousedown', handleDocumentClick)

onBeforeUnmount(() => {
  if (searchTimer) window.clearTimeout(searchTimer)
  document.removeEventListener('mousedown', handleDocumentClick)
})
</script>
