import { waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Filters } from '../api/types'
import { renderHookWithProviders, stubApi } from '../test/utils'
import { useKpis } from './useKpis'

const metric = { value: 100, previous: 50, deltaPct: 100 }
const kpis = { revenue: metric, rentals: metric, activeCustomers: metric, avgPayment: metric }
const filters: Filters = { from: '2022-07-01', to: '2022-07-10', store: 2, granularity: 'week' }

afterEach(() => vi.unstubAllGlobals())

describe('useKpis', () => {
  it('requests the filtered range and returns the figures', async () => {
    const calls = stubApi({ '/api/kpis': kpis })
    const { result } = renderHookWithProviders(() => useKpis(filters))
    await waitFor(() => expect(result.current.data).toEqual(kpis))
    expect(calls).toEqual(['/api/kpis?from=2022-07-01&to=2022-07-10&store=2'])
  })

  it('omits the store when none is chosen', async () => {
    const calls = stubApi({ '/api/kpis': kpis })
    const { result } = renderHookWithProviders(() => useKpis({ ...filters, store: null }))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(calls).toEqual(['/api/kpis?from=2022-07-01&to=2022-07-10'])
  })
})
