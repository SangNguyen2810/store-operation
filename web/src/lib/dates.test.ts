import { describe, expect, it } from 'vitest'
import { addDays, daysInclusive, isIsoDate } from './dates'

describe('isIsoDate', () => {
  it.each(['2022-07-27', '2020-02-29', '2000-01-01'])('accepts %s', (v) => expect(isIsoDate(v)).toBe(true))

  it.each([
    ['empty', ''],
    ['null', null],
    ['undefined', undefined],
    ['not a date', 'nope'],
    ['wrong order', '27-07-2022'],
    ['no padding', '2022-7-27'],
    ['month 13', '2022-13-01'],
    ['day 32', '2022-01-32'],
    ['Feb 30', '2022-02-30'],
    ['non-leap Feb 29', '2021-02-29'],
    ['with time', '2022-07-27T00:00:00Z'],
    ['trailing text', '2022-07-27x'],
  ])('rejects %s', (_name, v) => expect(isIsoDate(v)).toBe(false))
})

describe('addDays', () => {
  it.each([
    ['2022-07-27', 1, '2022-07-28'],
    ['2022-07-27', -29, '2022-06-28'],
    ['2022-07-31', 1, '2022-08-01'],
    ['2021-12-31', 1, '2022-01-01'],
    ['2022-01-01', -1, '2021-12-31'],
    ['2020-02-28', 1, '2020-02-29'],
    ['2021-02-28', 1, '2021-03-01'],
    ['2022-07-27', 0, '2022-07-27'],
  ])('%s %+d days is %s', (from, n, want) => expect(addDays(from, n)).toBe(want))
})

describe('daysInclusive', () => {
  it.each([
    ['2022-07-27', '2022-07-27', 1],
    ['2022-06-28', '2022-07-27', 30],
    ['2022-01-01', '2022-12-31', 365],
    ['2020-01-01', '2020-12-31', 366],
    ['2022-03-01', '2022-02-28', 0],
  ])('%s to %s is %d', (a, b, want) => expect(daysInclusive(a, b)).toBe(want))

  it('is the inverse of addDays', () => {
    expect(daysInclusive('2022-06-28', addDays('2022-06-28', 29))).toBe(30)
  })
})
