import { act, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderHookWithProviders } from '../test/utils'
import { useFilters } from './useFilters'

const ASOF = '2022-07-27'

describe('useFilters', () => {
  it('falls back to the last 30 days at the as-of date', () => {
    const { result } = renderHookWithProviders(() => useFilters(ASOF))
    expect(result.current[0]).toEqual({ from: '2022-06-28', to: ASOF, store: null, granularity: 'day' })
  })

  it('reads the filters from the URL', () => {
    const { result } = renderHookWithProviders(
      () => useFilters(ASOF),
      '/?from=2022-03-01&to=2022-03-31&store=1&granularity=week',
    )
    expect(result.current[0]).toEqual({ from: '2022-03-01', to: '2022-03-31', store: 1, granularity: 'week' })
  })

  it('writes changes back to the URL state', async () => {
    const { result } = renderHookWithProviders(() => useFilters(ASOF))
    act(() => result.current[1]({ ...result.current[0], store: 2 }))
    await waitFor(() => expect(result.current[0].store).toBe(2))
  })
})
