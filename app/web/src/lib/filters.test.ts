import { describe, expect, it } from 'vitest'
import type { DateRange } from '../api/types'
import { RANGES } from '../test/fixtures'
import { activePanelFilters, defaultFilters, defaultRange, findRange, parseFilters, rangeFilters, serializeFilters } from './filters'

const q = (s: string) => new URLSearchParams(s)
const DEFAULT = { from: '2022-06-28', to: '2022-07-27' }

describe('defaultRange', () => {
  it('is the one the API marks', () => {
    expect(defaultRange(RANGES).id).toBe('last-30')
    const moved = RANGES.map((r) => ({ ...r, default: r.id === 'last-7' }))
    expect(defaultRange(moved).id).toBe('last-7')
  })

  it('falls back to the first range when none is marked', () => {
    expect(defaultRange(RANGES.map((r) => ({ ...r, default: false }))).id).toBe('last-7')
  })

  it('throws, rather than guessing, when there are no ranges', () => {
    expect(() => defaultRange([])).toThrow(/no date ranges/)
  })
})

describe('findRange', () => {
  it('matches both ends exactly', () => {
    expect(findRange(RANGES, '2022-06-28', '2022-07-27')?.id).toBe('last-30')
    expect(findRange(RANGES, '2022-06-29', '2022-07-27')).toBeUndefined()
    expect(findRange(RANGES, '2022-06-28', '2022-07-26')).toBeUndefined()
  })
})

describe('parseFilters', () => {
  it('defaults to the range the API marks as default', () => {
    expect(parseFilters(q(''), RANGES)).toEqual({ ...DEFAULT, store: null, granularity: 'day' })
  })

  it('follows the default the API gives', () => {
    const ranges: DateRange[] = [{ ...RANGES[0], default: true }]
    expect(parseFilters(q(''), ranges)).toMatchObject({ from: '2022-07-21', to: '2022-07-27' })
  })

  it('reads every parameter', () => {
    expect(parseFilters(q('from=2022-03-01&to=2022-03-31&store=2&granularity=week'), RANGES)).toEqual({
      from: '2022-03-01',
      to: '2022-03-31',
      store: 2,
      granularity: 'week',
    })
  })

  it.each([
    ['unparseable dates', 'from=nope&to=2022-03-31'],
    ['a range that is not offered', 'from=2022-03-05&to=2022-03-20'],
    ['a range off by one day', 'from=2022-03-02&to=2022-03-31'],
    ['only one date', 'from=2022-03-01'],
    ['only the other date', 'to=2022-03-31'],
  ])('falls back to default dates for %s', (_name, qs) => {
    const f = parseFilters(q(qs), RANGES)
    expect([f.from, f.to]).toEqual([DEFAULT.from, DEFAULT.to])
  })

  it('drops a link to a range the API no longer offers', () => {
    const without = RANGES.filter((r) => r.id !== '2022-03')
    const f = parseFilters(q('from=2022-03-01&to=2022-03-31&store=1'), without)
    expect([f.from, f.to, f.store]).toEqual([DEFAULT.from, DEFAULT.to, 1])
  })

  it('ignores a bad store or granularity', () => {
    const f = parseFilters(q('store=abc&granularity=year'), RANGES)
    expect(f.store).toBeNull()
    expect(f.granularity).toBe('day')
    expect(parseFilters(q('store=-1'), RANGES).store).toBeNull()
  })
})

describe('serializeFilters', () => {
  it('writes nothing for the defaults', () => {
    expect(serializeFilters(defaultFilters(RANGES), RANGES).toString()).toBe('')
  })

  it('writes both dates for any other range', () => {
    const f = rangeFilters(defaultFilters(RANGES), RANGES[0])
    expect(serializeFilters(f, RANGES).toString()).toBe('from=2022-07-21&to=2022-07-27')
  })

  it('round-trips non-default filters', () => {
    const f = { from: '2022-03-01', to: '2022-03-31', store: 1, granularity: 'month' as const }
    expect(parseFilters(serializeFilters(f, RANGES), RANGES)).toEqual(f)
  })
})

describe('rangeFilters', () => {
  it('keeps store and granularity, replaces the range', () => {
    const cur = { from: '2022-01-01', to: '2022-01-31', store: 2, granularity: 'week' as const }
    expect(rangeFilters(cur, RANGES[0])).toEqual({ from: '2022-07-21', to: '2022-07-27', store: 2, granularity: 'week' })
  })
})

describe('activePanelFilters', () => {
  const base = defaultFilters(RANGES)
  const withRange = (id: string) => rangeFilters(base, RANGES.find((r) => r.id === id)!)

  it('counts nothing at the defaults', () => {
    expect(activePanelFilters(base, RANGES)).toBe(0)
  })

  it('does not count a range that has a button of its own', () => {
    expect(activePanelFilters(withRange('last-7'), RANGES)).toBe(0)
    expect(activePanelFilters(withRange('last-90'), RANGES)).toBe(0)
  })

  it('counts a range that is only in the menu', () => {
    expect(activePanelFilters(withRange('2022-q1'), RANGES)).toBe(1)
    expect(activePanelFilters(withRange('last-month'), RANGES)).toBe(1)
  })

  it('does not count the default range even when it has no button', () => {
    const ranges = RANGES.map((r) => ({ ...r, default: r.id === 'last-60' }))
    expect(activePanelFilters(defaultFilters(ranges), ranges)).toBe(0)
  })

  it('counts a store and a grouping', () => {
    expect(activePanelFilters({ ...base, store: 1 }, RANGES)).toBe(1)
    expect(activePanelFilters({ ...base, granularity: 'week' }, RANGES)).toBe(1)
    expect(activePanelFilters({ ...withRange('2022-q1'), store: 1, granularity: 'month' }, RANGES)).toBe(3)
  })
})
