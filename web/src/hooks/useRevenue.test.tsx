import { waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHookWithProviders, stubApi } from '../test/utils'
import { useRevenue } from './useRevenue'

const points = [{ bucket: '2022-07-04', storeId: 2, revenue: 12.5 }]
const filters = { from: "2022-07-01", to: "2022-07-10", store: 2, granularity: "week" as const }

afterEach(() => vi.unstubAllGlobals())

describe('useRevenue', () => {
  it('requests the range and granularity', async () => {
    const calls = stubApi({ '/api/revenue': points })
    const { result } = renderHookWithProviders(() => useRevenue(filters))
    await waitFor(() => expect(result.current.data).toEqual(points))
    expect(calls).toEqual(['/api/revenue?from=2022-07-01&to=2022-07-10&store=2&granularity=week'])
  })

  it('keeps the previous data on screen while new filters load', async () => {
    stubApi({ '/api/revenue': points })
    let current = filters
    const { result, rerender } = renderHookWithProviders(() => useRevenue(current))
    await waitFor(() => expect(result.current.data).toEqual(points))

    current = { ...filters, store: null } // a new cache key, so the query starts loading
    rerender()
    expect(result.current.isPlaceholderData).toBe(true)
    expect(result.current.data).toEqual(points)
    await waitFor(() => expect(result.current.isPlaceholderData).toBe(false))
  })
})
