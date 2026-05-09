import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SettlementPoolsView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

describe('Admin SettlementPoolsView history selection', () => {
  it('renders selected cycle snapshots instead of only the active estimate', () => {
    expect(viewSource).toContain(':display-estimate="displayEstimate"')
    expect(viewSource).toContain('const selectedCycleId = ref<number | null>(null)')
    expect(viewSource).toContain('return summary.value.cycles.find(cycle => cycle.id === selectedCycleId.value)?.snapshot ?? null')
  })

  it('adds a view action to the cycles table', () => {
    expect(viewSource).toContain('{ key: \'actions\', label: \'\', class: rightAlignedColumnClass }')
    expect(viewSource).toContain('@click="selectCycle(row.cycle_id)"')
    expect(viewSource).toContain('function selectCycle(cycleId: number | null)')
  })

  it('separates long-term candidates from current-cycle participants', () => {
    expect(viewSource).toContain('settlementPools.candidates')
    expect(viewSource).toContain('settlementPools.currentParticipants')
    expect(viewSource).toContain('settlementPoolsAPI.syncCandidates')
    expect(viewSource).toContain('settlementPoolsAPI.forceJoinCurrentCycle')
    expect(viewSource).toContain('settlementPoolsAPI.removeCurrentParticipant')
    expect(viewSource).not.toContain('syncParticipants')
  })

  it('shows current-cycle account usage', () => {
    expect(viewSource).toContain('settlementPools.accountUsage')
    expect(viewSource).toContain('const accountUsageRows = computed<SettlementPoolAccountUsage[]>')
    expect(viewSource).toContain('return displayEstimate.value?.account_usage || []')
  })
})
