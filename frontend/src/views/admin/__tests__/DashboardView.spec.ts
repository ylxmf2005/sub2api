import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { DashboardStats } from '@/types'
import DashboardView from '../DashboardView.vue'

const {
  adminGetSnapshotV2,
  adminGetUserUsageTrend,
  adminGetUserSpendingRanking,
  adminGetAllGroups,
  monitorGetSnapshotV2,
  monitorGetUserUsageTrend,
  monitorGetUserSpendingRanking,
  monitorGetAllGroups,
  routeState,
  push
} = vi.hoisted(() => ({
  adminGetSnapshotV2: vi.fn(),
  adminGetUserUsageTrend: vi.fn(),
  adminGetUserSpendingRanking: vi.fn(),
  adminGetAllGroups: vi.fn(),
  monitorGetSnapshotV2: vi.fn(),
  monitorGetUserUsageTrend: vi.fn(),
  monitorGetUserSpendingRanking: vi.fn(),
  monitorGetAllGroups: vi.fn(),
  routeState: { path: '/admin/dashboard' },
  push: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getSnapshotV2: adminGetSnapshotV2,
      getUserUsageTrend: adminGetUserUsageTrend,
      getUserSpendingRanking: adminGetUserSpendingRanking
    },
    groups: {
      getAll: adminGetAllGroups
    }
  }
}))

vi.mock('@/api/monitor', () => ({
  default: {
    dashboard: {
      getSnapshotV2: monitorGetSnapshotV2,
      getUserUsageTrend: monitorGetUserUsageTrend,
      getUserSpendingRanking: monitorGetUserSpendingRanking
    },
    groups: {
      getAll: monitorGetAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const createDashboardStats = (): DashboardStats => ({
  total_users: 0,
  today_new_users: 0,
  active_users: 0,
  hourly_active_users: 0,
  stats_updated_at: '',
  stats_stale: false,
  total_api_keys: 0,
  active_api_keys: 0,
  total_accounts: 0,
  normal_accounts: 0,
  error_accounts: 0,
  ratelimit_accounts: 0,
  overload_accounts: 0,
  total_requests: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_creation_tokens: 0,
  total_cache_read_tokens: 0,
  total_tokens: 0,
  total_cost: 0,
  total_actual_cost: 0,
  total_account_cost: 0,
  today_requests: 0,
  today_input_tokens: 0,
  today_output_tokens: 0,
  today_cache_creation_tokens: 0,
  today_cache_read_tokens: 0,
  today_tokens: 0,
  today_cost: 0,
  today_actual_cost: 0,
  today_account_cost: 0,
  average_duration_ms: 0,
  uptime: 0,
  rpm: 0,
  tpm: 0
})

const selectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  methods: {
    normalizeValue(value: string) {
      if (value === '') return null
      return /^-?\d+$/.test(value) ? Number.parseInt(value, 10) : value
    }
  },
  template: `
    <select
      :value="modelValue ?? ''"
      @change="$emit('update:modelValue', normalizeValue($event.target.value)); $emit('change', normalizeValue($event.target.value))"
    >
      <option v-for="option in options" :key="String(option.value ?? '')" :value="option.value ?? ''">
        {{ option.label }}
      </option>
    </select>
  `
}

const mountDashboard = () =>
  mount(DashboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        LoadingSpinner: true,
        Icon: true,
        DateRangePicker: true,
        Select: selectStub,
        ModelDistributionChart: true,
        TokenUsageTrend: true,
        Line: true
      }
    }
  })

describe('admin DashboardView', () => {
  beforeEach(() => {
    routeState.path = '/admin/dashboard'
    push.mockReset()

    adminGetSnapshotV2.mockReset()
    adminGetUserUsageTrend.mockReset()
    adminGetUserSpendingRanking.mockReset()
    adminGetAllGroups.mockReset()
    monitorGetSnapshotV2.mockReset()
    monitorGetUserUsageTrend.mockReset()
    monitorGetUserSpendingRanking.mockReset()
    monitorGetAllGroups.mockReset()

    adminGetSnapshotV2.mockResolvedValue({
      stats: createDashboardStats(),
      trend: [],
      models: []
    })
    adminGetUserUsageTrend.mockResolvedValue({
      trend: [],
      start_date: '',
      end_date: '',
      granularity: 'hour'
    })
    adminGetUserSpendingRanking.mockResolvedValue({
      ranking: [],
      total_actual_cost: 0,
      total_requests: 0,
      total_tokens: 0,
      start_date: '',
      end_date: ''
    })
    adminGetAllGroups.mockResolvedValue([
      { id: 2, name: 'OpenAI Shared', platform: 'openai' }
    ])

    monitorGetSnapshotV2.mockResolvedValue({
      stats: createDashboardStats(),
      trend: [],
      models: []
    })
    monitorGetUserUsageTrend.mockResolvedValue({
      trend: [],
      start_date: '',
      end_date: '',
      granularity: 'hour'
    })
    monitorGetUserSpendingRanking.mockResolvedValue({
      ranking: [],
      total_actual_cost: 0,
      total_requests: 0,
      total_tokens: 0,
      start_date: '',
      end_date: ''
    })
    monitorGetAllGroups.mockResolvedValue([
      { id: 2, name: 'OpenAI Shared', platform: 'openai' }
    ])
  })

  it('uses last 24 hours as default dashboard range', async () => {
    mountDashboard()

    await flushPromises()

    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)

    expect(adminGetSnapshotV2).toHaveBeenCalledTimes(1)
    expect(adminGetSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour'
    }))
  })

  it('reloads admin dashboard charts with group filter when a group is selected', async () => {
    const wrapper = mountDashboard()

    await flushPromises()

    adminGetSnapshotV2.mockClear()
    adminGetUserUsageTrend.mockClear()
    adminGetUserSpendingRanking.mockClear()

    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(2)

    await selects[0].setValue('2')
    await flushPromises()

    expect(adminGetSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 2
    }))
    expect(adminGetUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 2
    }))
    expect(adminGetUserSpendingRanking).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 2
    }))
  })

  it('uses monitor endpoints when rendered on /monitor', async () => {
    routeState.path = '/monitor'
    const wrapper = mountDashboard()

    await flushPromises()

    expect(monitorGetSnapshotV2).toHaveBeenCalledTimes(1)
    expect(monitorGetUserUsageTrend).toHaveBeenCalledTimes(1)
    expect(monitorGetUserSpendingRanking).toHaveBeenCalledTimes(1)
    expect(monitorGetAllGroups).toHaveBeenCalledTimes(1)

    monitorGetSnapshotV2.mockClear()
    monitorGetUserUsageTrend.mockClear()
    monitorGetUserSpendingRanking.mockClear()

    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(2)

    await selects[0].setValue('2')
    await flushPromises()

    expect(monitorGetSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 2
    }))
    expect(monitorGetUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 2
    }))
    expect(monitorGetUserSpendingRanking).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 2
    }))
  })
})
