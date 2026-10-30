import { describe, expect, it } from 'vitest'
import { makeStore } from '../../test/fixtures'
import { pivotRevenue, totalRevenue } from './pivot'

const points = [
  { bucket: '2022-07-01', storeId: 1, revenue: 10 },
  { bucket: '2022-07-01', storeId: 2, revenue: 20 },
  { bucket: '2022-07-02', storeId: 1, revenue: 0 },
  { bucket: '2022-07-02', storeId: 2, revenue: 5 },
]

describe('pivotRevenue', () => {
  it('makes one row per bucket with a column per store', () => {
    const { rows, series } = pivotRevenue(points, [makeStore(1, 'Boksburg')])
    expect(rows).toEqual([
      { x: '2022-07-01', s1: 10, s2: 20 },
      { x: '2022-07-02', s1: 0, s2: 5 },
    ])
    expect(series).toEqual([
      { key: 's1', label: 'Boksburg' },
      { key: 's2', label: 'Store 2' },
    ])
  })
  it('sums revenue', () => expect(totalRevenue(points)).toBe(35))
})
