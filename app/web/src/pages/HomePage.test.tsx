import { screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders, stubApi } from '../test/utils'
import { HomePage } from './HomePage'
import { RANGES } from '../test/fixtures'

afterEach(() => vi.unstubAllGlobals())

describe('HomePage', () => {
  it('is the dashboard', async () => {
    stubApi({ '/api/meta': { asOf: '2022-07-27' }, '/api/presets': RANGES, '/api/stores': [] })
    renderWithProviders(<HomePage />)
    expect(await screen.findByRole('heading', { name: 'Store Operations' })).toBeInTheDocument()
  })
})
