import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Filters, RevenuePoint, Store } from '../../api/types'
import { STORES } from '../../test/fixtures'
import { RevenueChart } from './RevenueChart'

const filters: Filters = { from: '2022-07-26', to: '2022-07-27', store: null, granularity: 'day' }
const stores: Store[] = STORES
const points: RevenuePoint[] = [
  { bucket: '2022-07-26', storeId: 1, revenue: 100 },
  { bucket: '2022-07-26', storeId: 2, revenue: 50.5 },
  { bucket: '2022-07-27', storeId: 1, revenue: 10 },
  { bucket: '2022-07-27', storeId: 2, revenue: 0 },
]

describe('RevenueChart', () => {
  it('describes the chart with the period, the total and the stores', () => {
    render(<RevenueChart points={points} stores={stores} filters={filters} />)
    expect(screen.getByRole('img')).toHaveAccessibleName(
      'Revenue per day, Jul 26 to Jul 27: $160.50 in total across Boksburg, South Africa and Hamilton, New Zealand.',
    )
  })

  it('names the granularity', () => {
    render(<RevenueChart points={points} stores={stores} filters={{ ...filters, granularity: 'month' }} />)
    expect(screen.getByRole('img').getAttribute('aria-label')).toMatch(/^Revenue per month,/)
  })

  it('shows one table column per store, with a fallback name before the stores load', () => {
    render(<RevenueChart points={points} stores={[]} filters={filters} />)
    expect(screen.getByRole('columnheader', { name: 'Store 1' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Store 2' })).toBeInTheDocument()
  })
})
