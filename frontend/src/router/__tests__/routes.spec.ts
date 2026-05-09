import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('route ownership', () => {
  it('keeps /monitor reserved for the mapped Monitor dashboard', () => {
    const monitorRoutes = router.getRoutes().filter(route => route.path === '/monitor')

    expect(monitorRoutes).toHaveLength(1)
    expect(monitorRoutes[0].name).toBe('Monitor')
    expect(router.resolve('/monitor').name).toBe('Monitor')
  })

  it('uses a dedicated route for user-facing channel status', () => {
    const channelStatusRoutes = router.getRoutes().filter(route => route.path === '/channel-status')

    expect(channelStatusRoutes).toHaveLength(1)
    expect(channelStatusRoutes[0].name).toBe('ChannelStatus')
    expect(router.resolve('/channel-status').name).toBe('ChannelStatus')
  })

  it('registers an authenticated ccgo device approval route', () => {
    const route = router.getRoutes().find(route => route.path === '/ccgo/device')

    expect(route?.name).toBe('CcgoDeviceLogin')
    expect(route?.meta.requiresAuth).toBe(true)
    expect(router.resolve('/ccgo/device?code=ABCD-EFGH').name).toBe('CcgoDeviceLogin')
  })
})
