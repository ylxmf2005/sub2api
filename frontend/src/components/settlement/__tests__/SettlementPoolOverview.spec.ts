import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SettlementPoolOverview.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('SettlementPoolOverview card summaries', () => {
  it('does not render tier-rate metadata under overview cards', () => {
    expect(componentSource).not.toContain('card.meta')
    expect(componentSource).not.toContain('tierSummary')
    expect(componentSource).not.toContain('tierText(')
  })

  it('can render an explicitly selected cycle estimate', () => {
    expect(componentSource).toContain('displayEstimate?: SettlementPoolEstimate | null')
    expect(componentSource).toContain('props.displayEstimate !== undefined')
    expect(componentSource).toContain('estimate.value?.total_cost ?? props.summary.active_cycle?.total_cost')
  })
})
