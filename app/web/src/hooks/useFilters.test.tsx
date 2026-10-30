import { act, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { RANGES } from '../test/fixtures'
import { renderHookWithProviders } from '../test/utils'
import { useFilters } from './useFilters'

describe('useFilters', () => {
  it('falls back to the default range', () => {
    const { result } = renderHookWithProviders(() => useFilters(RANGES))
    expect(result.current[0]).toEqual({ from: '2022-06-28', to: '2022-07-27', store: null, granularity: 'day' })
  })

  it('reads the filters from the URL', () => {
    const { result } = renderHookWithProviders(
      () => useFilters(RANGES),
      '/?from=2022-03-01&to=2022-03-31&store=1&granularity=week',
    )
    expect(result.current[0]).toEqual({ from: '2022-03-01', to: '2022-03-31', store: 1, granularity: 'week' })
  })

  it('ignores a range that is not offered', () => {
    const { result } = renderHookWithProviders(() => useFilters(RANGES), '/?from=2022-03-05&to=2022-03-20')
    expect(result.current[0]).toMatchObject({ from: '2022-06-28', to: '2022-07-27' })
  })

  it('writes changes back to the URL state', async () => {
    const { result } = renderHookWithProviders(() => useFilters(RANGES))
    act(() => result.current[1]({ ...result.current[0], store: 2 }))
    await waitFor(() => expect(result.current[0].store).toBe(2))
  })
})
