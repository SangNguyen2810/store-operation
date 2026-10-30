import { describe, expect, it } from 'vitest'
import { formatBucket, formatCurrency, formatCurrencyWhole, formatDelta, formatInteger } from './format'

describe('format', () => {
  it('formats money and counts', () => {
    expect(formatCurrency(10707.2)).toBe('$10,707.20')
    expect(formatInteger(4158)).toBe('4,158')
  })
  it('drops the cents for axis ticks', () => {
    expect(formatCurrencyWhole(10707.6)).toBe('$10,708')
    expect(formatCurrencyWhole(0)).toBe('$0')
  })
  it('signs deltas', () => {
    expect(formatDelta(3.25)).toBe('+3.3%')
    expect(formatDelta(-2.43)).toBe('-2.4%')
    expect(formatDelta(0)).toBe('0.0%')
  })
  it('formats buckets without timezone drift', () => {
    expect(formatBucket('2022-07-01')).toBe('Jul 1')
  })
})
