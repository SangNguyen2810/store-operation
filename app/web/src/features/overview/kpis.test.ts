import { describe, expect, it } from 'vitest'
import type { Kpis } from '../../api/types'
import { toKpiItems, toTrend } from './kpis'

const kpis: Kpis = {
  revenue: { value: 10707.22, previous: 10973.36, deltaPct: -2.43 },
  rentals: { value: 4158, previous: 2778, deltaPct: 49.68 },
  activeCustomers: { value: 598, previous: 0, deltaPct: null },
  avgPayment: { value: 4.15, previous: 4.15, deltaPct: 0 },
}

describe('toTrend', () => {
  it('describes a fall, a rise and no change in words', () => {
    expect(toTrend({ value: 1, previous: 1, deltaPct: -2.43 })).toEqual({
      direction: 'down',
      text: '-2.4%',
      spokenLabel: 'down 2.4% versus previous period',
    })
    expect(toTrend({ value: 1, previous: 1, deltaPct: 49.68 })).toMatchObject({ direction: 'up', text: '+49.7%' })
    expect(toTrend({ value: 1, previous: 1, deltaPct: 0 })).toMatchObject({
      direction: 'flat',
      spokenLabel: 'unchanged 0.0% versus previous period',
    })
  })

  it('is null when there is no previous value', () => {
    expect(toTrend({ value: 5, previous: 0, deltaPct: null })).toBeNull()
  })
})

describe('toKpiItems', () => {
  it('returns the four cards in order with formatted values and captions', () => {
    const items = toKpiItems(kpis)
    expect(items.map((i) => i.id)).toEqual(['revenue', 'rentals', 'activeCustomers', 'avgPayment'])
    expect(items[0]).toMatchObject({ label: 'Revenue', value: '$10,707.22', caption: 'prev. $10,973.36' })
    expect(items[1]).toMatchObject({ value: '4,158', caption: 'prev. 2,778' })
    expect(items[2].trend).toBeNull()
    expect(items[3]).toMatchObject({ value: '$4.15', caption: 'prev. $4.15' })
  })
})
