import { describe, expect, it } from 'vitest'
import type { Filters } from '../api/types'
import { queryKeys } from './queryKeys'

const base: Filters = { from: '2022-07-01', to: '2022-07-10', store: 1, granularity: 'day' }

// A key that ignores a filter would serve one filter's data for another.
describe('queryKeys', () => {
  it('are equal for equal filters, so the cache is shared', () => {
    expect(queryKeys.kpis({ ...base })).toEqual(queryKeys.kpis(base))
    expect(queryKeys.revenue({ ...base })).toEqual(queryKeys.revenue(base))
  })

  it.each([
    ['from', { from: '2022-06-01' }],
    ['to', { to: '2022-07-11' }],
    ['store', { store: 2 }],
    ['all stores', { store: null }],
  ] as const)('every data key changes with %s', (_name, change) => {
    const other = { ...base, ...change }
    expect(queryKeys.kpis(other)).not.toEqual(queryKeys.kpis(base))
    expect(queryKeys.revenue(other)).not.toEqual(queryKeys.revenue(base))
    expect(queryKeys.topFilms(other)).not.toEqual(queryKeys.topFilms(base))
  })

  it('only revenue depends on the granularity', () => {
    const week = { ...base, granularity: 'week' as const }
    expect(queryKeys.revenue(week)).not.toEqual(queryKeys.revenue(base))
    expect(queryKeys.kpis(week)).toEqual(queryKeys.kpis(base))
    expect(queryKeys.topFilms(week)).toEqual(queryKeys.topFilms(base))
  })

  it('keeps each resource in its own namespace', () => {
    const keys = [queryKeys.meta, queryKeys.stores, queryKeys.kpis(base), queryKeys.revenue(base), queryKeys.topFilms(base)]
    expect(new Set(keys.map((k) => k[0])).size).toBe(keys.length)
  })
})
