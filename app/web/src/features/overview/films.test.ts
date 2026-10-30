import { describe, expect, it } from 'vitest'
import { toFilmRows } from './films'

const films = [
  { filmId: 7, title: 'FOOL MOCKINGBIRD', rentals: 1234, revenue: 52.93 },
  { filmId: 9, title: 'DORADO NOTTING', rentals: 8, revenue: 26.465 },
]

describe('toFilmRows', () => {
  it('formats the figures and keeps the order', () => {
    const rows = toFilmRows(films)
    expect(rows.map((r) => r.id)).toEqual([7, 9])
    expect(rows[0]).toMatchObject({ label: 'FOOL MOCKINGBIRD', meta: '1,234 rentals', values: ['1,234', '$52.93'] })
  })

  it('draws each bar relative to the highest revenue', () => {
    const rows = toFilmRows(films)
    expect(rows[0].bar).toBe(1)
    expect(rows[1].bar).toBeCloseTo(0.5)
  })

  it('handles an empty list and all-zero revenue without dividing by zero', () => {
    expect(toFilmRows([])).toEqual([])
    expect(toFilmRows([{ filmId: 1, title: 'A', rentals: 0, revenue: 0 }])[0].bar).toBe(0)
  })
})
