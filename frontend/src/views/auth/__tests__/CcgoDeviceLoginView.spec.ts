import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CcgoDeviceLoginView from '@/views/auth/CcgoDeviceLoginView.vue'

const { routeState, apiPostMock } = vi.hoisted(() => ({
  routeState: {
    query: {} as Record<string, unknown>,
  },
  apiPostMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post: (...args: any[]) => apiPostMock(...args),
  },
}))

describe('CcgoDeviceLoginView', () => {
  beforeEach(() => {
    routeState.query = {}
    apiPostMock.mockReset()
  })

  it('prefills the code from the route query and approves the login', async () => {
    routeState.query = { code: 'abcd-efgh' }
    apiPostMock.mockResolvedValue({ status: 'approved' })

    const wrapper = mount(CcgoDeviceLoginView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /></div>' },
          Icon: true,
        },
      },
    })
    await flushPromises()

    const input = wrapper.find('input')
    expect((input.element as HTMLInputElement).value).toBe('ABCD-EFGH')

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(apiPostMock).toHaveBeenCalledWith('/ccgo/device-login/approve', {
      user_code: 'ABCD-EFGH',
    })
    expect(wrapper.text()).toContain('ccgo login approved')
  })

  it('disables authorization until the code has 8 characters', async () => {
    const wrapper = mount(CcgoDeviceLoginView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /></div>' },
          Icon: true,
        },
      },
    })

    await wrapper.find('input').setValue('ABC')
    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    await flushPromises()

    expect(apiPostMock).not.toHaveBeenCalled()
  })
})
