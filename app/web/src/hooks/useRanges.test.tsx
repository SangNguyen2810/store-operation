import { waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RANGES } from '../test/fixtures'
import { renderHookWithProviders, stubApi } from '../test/utils'
import { useRanges } from './useRanges'

afterEach(() => vi.unstubAllGlobals())

describe('useRanges', () => {
  it('loads the ranges the API offers', async () => {
    const calls = stubApi({ '/api/presets': RANGES })
    const { result } = renderHookWithProviders(() => useRanges())
    await waitFor(() => expect(result.current.data).toEqual(RANGES))
    expect(calls).toEqual(['/api/presets'])
  })

  it('reports a failure', async () => {
    stubApi({})
    const { result } = renderHookWithProviders(() => useRanges())
    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
