import { waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHookWithProviders, stubApi } from '../test/utils'
import { useStores } from './useStores'

afterEach(() => vi.unstubAllGlobals())

describe('useStores', () => {
  it('loads the stores', async () => {
    const calls = stubApi({ '/api/stores': [{ id: 1, label: 'Boksburg, South Africa' }] })
    const { result } = renderHookWithProviders(() => useStores())
    await waitFor(() => expect(result.current.data).toEqual([{ id: 1, label: 'Boksburg, South Africa' }]))
    expect(calls).toEqual(['/api/stores'])
  })
})
