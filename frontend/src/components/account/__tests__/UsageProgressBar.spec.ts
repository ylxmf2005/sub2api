import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UsageProgressBar from '../UsageProgressBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('UsageProgressBar', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-17T00:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('showNowWhenIdle=true 且利用率为 0 时显示“现在”', () => {
    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '5h',
        utilization: 0,
        resetsAt: '2026-03-17T02:30:00Z',
        showNowWhenIdle: true,
        color: 'indigo'
      }
    })

    expect(wrapper.text()).toContain('现在')
    expect(wrapper.text()).not.toContain('2h 30m')
  })

  it('showNowWhenIdle=true 但利用率大于 0 时显示倒计时', () => {
    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '7d',
        utilization: 12,
        resetsAt: '2026-03-17T02:30:00Z',
        showNowWhenIdle: true,
        color: 'emerald'
      }
    })

    expect(wrapper.text()).toContain('2h 30m')
    expect(wrapper.text()).not.toContain('现在')
  })

  it('showNowWhenIdle=false 时保持原有倒计时行为', () => {
    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '1d',
        utilization: 0,
        resetsAt: '2026-03-17T02:30:00Z',
        showNowWhenIdle: false,
        color: 'indigo'
      }
    })

    expect(wrapper.text()).toContain('2h 30m')
    expect(wrapper.text()).not.toContain('现在')
  })

  it('按窗口成本除以使用率显示本周预估可用额度', () => {
    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '7d',
        utilization: 20,
        color: 'emerald',
        showEstimatedTotal: true,
        windowStats: {
          requests: 10,
          tokens: 1000,
          cost: 4,
          standard_cost: 4,
          user_cost: 4
        }
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.usageWindow.estimatedTotal')
    expect(wrapper.text()).toContain('$20.00')
  })

  it('没有有效百分比或成本时不显示本周预估可用额度', () => {
    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '7d',
        utilization: 0,
        color: 'emerald',
        showEstimatedTotal: true,
        windowStats: {
          requests: 0,
          tokens: 0,
          cost: 4,
          standard_cost: 4,
          user_cost: 4
        }
      }
    })

    expect(wrapper.text()).not.toContain('admin.accounts.usageWindow.estimatedTotal')
  })

  it('重置时间已经过期时不显示本周期预估总用量', () => {
    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '7d',
        utilization: 20,
        resetsAt: '2026-03-16T23:00:00Z',
        color: 'emerald',
        showEstimatedTotal: true,
        windowStats: {
          requests: 10,
          tokens: 1000,
          cost: 4,
          standard_cost: 4,
          user_cost: 4
        }
      }
    })

    expect(wrapper.text()).not.toContain('admin.accounts.usageWindow.estimatedTotal')
  })
})
