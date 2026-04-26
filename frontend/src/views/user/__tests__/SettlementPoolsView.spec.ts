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
})
