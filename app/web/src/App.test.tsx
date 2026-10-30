import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { stubApi } from './test/utils'
import { RANGES } from './test/fixtures'

afterEach(() => {
  vi.unstubAllGlobals()
  window.history.pushState({}, '', '/')
})

// App owns the real browser router and the app-wide query client.
describe('App', () => {
  it('renders the dashboard at /', async () => {
    stubApi({ '/api/meta': { asOf: '2022-07-27' }, '/api/presets': RANGES, '/api/stores': [] })
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Store Operations' })).toBeInTheDocument()
    expect(screen.getByRole('main')).toBeInTheDocument()
  })

  it('renders the 404 page for an unknown address', () => {
    window.history.pushState({}, '', '/nope')
    render(<App />)
    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
  })
})
