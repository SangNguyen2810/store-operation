import { waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHookWithProviders, stubApi } from '../test/utils'
import { useMeta } from './useMeta'

afterEach(() => vi.unstubAllGlobals())

describe('useMeta', () => {
  it('loads the as-of date', async () => {
    const calls = stubApi({ '/api/meta': { asOf: '2022-07-27' } })
    const { result } = renderHookWithProviders(() => useMeta())
    await waitFor(() => expect(result.current.data).toEqual({ asOf: '2022-07-27' }))
    expect(calls).toEqual(['/api/meta'])
  })

  it('reports an error when the API fails', async () => {
    stubApi({})
    const { result } = renderHookWithProviders(() => useMeta())
    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
