import { describe, expect, it } from 'vitest'
import { defaultFilters, parseFilters, presetFilters, serializeFilters } from './filters'

const ASOF = '2022-07-27'
const q = (s: string) => new URLSearchParams(s)

describe('parseFilters', () => {
  it('defaults to the 30 days ending at the as-of date', () => {
    expect(parseFilters(q(''), ASOF)).toEqual({
      from: '2022-06-28',
      to: '2022-07-27',
      store: null,
      granularity: 'day',
    })
  })

  it('reads every parameter', () => {
    expect(parseFilters(q('from=2022-03-01&to=2022-03-31&store=2&granularity=week'), ASOF)).toEqual({
      from: '2022-03-01',
      to: '2022-03-31',
      store: 2,
      granularity: 'week',
    })
  })

  it.each([
    ['unparseable dates', 'from=nope&to=2022-03-31'],
    ['impossible dates', 'from=2022-02-30&to=2022-03-31'],
    ['reversed range', 'from=2022-04-01&to=2022-03-01'],
    ['only one date', 'from=2022-03-01'],
    ['range over a year', 'from=2020-01-01&to=2022-01-01'],
  ])('falls back to default dates for %s', (_name, qs) => {
    const f = parseFilters(q(qs), ASOF)
    expect([f.from, f.to]).toEqual(['2022-06-28', '2022-07-27'])
  })

  it('ignores a bad store or granularity', () => {
    const f = parseFilters(q('store=abc&granularity=year'), ASOF)
    expect(f.store).toBeNull()
    expect(f.granularity).toBe('day')
    expect(parseFilters(q('store=-1'), ASOF).store).toBeNull()
  })
})

describe('serializeFilters', () => {
  it('writes nothing for the defaults', () => {
    expect(serializeFilters(defaultFilters(ASOF), ASOF).toString()).toBe('')
  })

  it('round-trips non-default filters', () => {
    const f = { from: '2022-03-01', to: '2022-03-31', store: 1, granularity: 'month' as const }
    expect(parseFilters(serializeFilters(f, ASOF), ASOF)).toEqual(f)
  })
})

describe('presetFilters', () => {
  it('keeps store and granularity, resets the range', () => {
    const cur = { from: '2022-01-01', to: '2022-01-31', store: 2, granularity: 'week' as const }
    expect(presetFilters(cur, ASOF, 7)).toEqual({
      from: '2022-07-21',
      to: '2022-07-27',
      store: 2,
      granularity: 'week',
    })
  })
})
