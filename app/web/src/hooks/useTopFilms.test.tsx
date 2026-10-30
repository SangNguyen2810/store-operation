import { waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Filters } from '../api/types'
import { renderHookWithProviders, stubApi } from '../test/utils'
import { useTopFilms } from './useTopFilms'

const films = [{ filmId: 1, title: 'ACADEMY DINOSAUR', rentals: 3, revenue: 12.5 }]
const filters: Filters = { from: '2022-07-01', to: '2022-07-10', store: 2, granularity: 'week' }

afterEach(() => vi.unstubAllGlobals())

describe('useTopFilms', () => {
  it('asks for the top 10 in the range', async () => {
    const calls = stubApi({ '/api/top/films': films })
    const { result } = renderHookWithProviders(() => useTopFilms(filters))
    await waitFor(() => expect(result.current.data).toEqual(films))
    expect(calls).toEqual(['/api/top/films?from=2022-07-01&to=2022-07-10&store=2&limit=10'])
  })
})
