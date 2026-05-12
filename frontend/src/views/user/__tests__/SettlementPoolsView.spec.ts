import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SettlementPoolsView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

describe('User SettlementPoolsView history selection', () => {
  it('passes the selected cycle estimate into the overview', () => {
    expect(viewSource).toContain(':display-estimate="displayEstimate(summary)"')
    expect(viewSource).toContain('function displayEstimate(summary: SettlementPoolSummary): SettlementPoolEstimate | null')
    expect(viewSource).toContain('return summaryCycles(summary).find(cycle => cycle.id === cycleId)?.snapshot ?? null')
  })

  it('exposes joining the current cycle instead of exiting it', () => {
    expect(viewSource).toContain('@click="joinCurrentCycle(summary)"')
    expect(viewSource).toContain('settlementPoolsAPI.joinCurrentCycle(groupId)')
    expect(viewSource).toContain('summary.can_join_active_cycle')
    expect(viewSource).not.toContain('can_exit_active_cycle')
  })

  it('shows current-cycle account usage for eligible users', () => {
    expect(viewSource).toContain('settlementPools.accountUsage')
    expect(viewSource).toContain('settlementPools.accountUsagePeriod')
    expect(viewSource).toContain('{ key: \'weekly_total_usage\', label: t(\'settlementPools.weeklyUsage\'), class: rightAlignedColumnClass }')
    expect(viewSource).toContain('{ key: \'manual_usage\', label: t(\'settlementPools.manualUsage\'), class: rightAlignedColumnClass }')
    expect(viewSource).toContain('function accountUsageRows(summary: SettlementPoolSummary): SettlementPoolAccountUsage[]')
    expect(viewSource).toContain('return displayEstimate(summary)?.account_usage || []')
    expect(viewSource).toContain('function settlementCyclePeriod(summary: SettlementPoolSummary): string')
  })
})
