import { describe, expect, it } from 'vitest'
import { addDays, daysInclusive, isIsoDate } from './dates'
import { formatBucket, formatCurrency, formatDelta, formatInteger } from './format'

describe('format', () => {
  it('formats money and counts', () => {
    expect(formatCurrency(10707.2)).toBe('$10,707.20')
    expect(formatInteger(4158)).toBe('4,158')
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

describe('dates', () => {
  it('validates ISO dates', () => {
    expect(isIsoDate('2022-02-28')).toBe(true)
    expect(isIsoDate('2022-02-30')).toBe(false)
    expect(isIsoDate('22-02-28')).toBe(false)
    expect(isIsoDate('')).toBe(false)
    expect(isIsoDate(null)).toBe(false)
  })
  it('adds days across month and year boundaries', () => {
    expect(addDays('2022-03-01', -1)).toBe('2022-02-28')
    expect(addDays('2022-01-01', -1)).toBe('2021-12-31')
  })
  it('counts inclusive days', () => {
    expect(daysInclusive('2022-07-01', '2022-07-01')).toBe(1)
    expect(daysInclusive('2022-06-28', '2022-07-27')).toBe(30)
  })
})
