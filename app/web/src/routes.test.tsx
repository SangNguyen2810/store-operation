import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppRoutes } from './routes'
import { RANGES, makeStore } from './test/fixtures'
import { renderWithProviders } from './test/utils'

function mockApi() {
  const json = (body: unknown) => new Response(JSON.stringify(body), { status: 200 })
  const kpi = { value: 100, previous: 50, deltaPct: 100 }
  const routes: Record<string, unknown> = {
    '/api/meta': { asOf: '2022-07-27' },
    '/api/presets': RANGES,
    '/api/stores': [makeStore(1, 'Boksburg, South Africa')],
    '/api/kpis': { revenue: kpi, rentals: kpi, activeCustomers: kpi, avgPayment: kpi },
    '/api/revenue': [{ bucket: '2022-07-27', storeId: 1, revenue: 100 }],
    '/api/top/films': [{ filmId: 1, title: 'ACADEMY DINOSAUR', rentals: 3, revenue: 12.5 }],
  }
  vi.stubGlobal('fetch', (input: string) =>
    Promise.resolve(routes[input.split('?')[0]] ? json(routes[input.split('?')[0]]) : new Response('{}', { status: 404 })),
  )
}

afterEach(() => vi.unstubAllGlobals())

describe('AppRoutes', () => {
  it('renders the dashboard at /', async () => {
    mockApi()
    renderWithProviders(<AppRoutes />, '/')
    expect(await screen.findByRole('heading', { name: 'Store Operations' })).toBeInTheDocument()
    expect(screen.queryByText('Page not found')).not.toBeInTheDocument()
  })

  it('keeps filters in the query string on /', async () => {
    mockApi()
    renderWithProviders(<AppRoutes />, '/?store=1')
    const store = await screen.findByLabelText('Store')
    await waitFor(() => expect(store).toHaveValue('1')) // options arrive with /api/stores
  })

  it.each(['/nope', '/api-docs', '/a/b/c'])('renders the 404 page at %s', (path) => {
    renderWithProviders(<AppRoutes />, path)
    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Store Operations' })).not.toBeInTheDocument()
  })

  it('the 404 link goes back to the dashboard', async () => {
    mockApi()
    renderWithProviders(<AppRoutes />, '/nope')
    await userEvent.click(screen.getByRole('link', { name: 'Back to the dashboard' }))
    expect(await screen.findByRole('heading', { name: 'Store Operations' })).toBeInTheDocument()
  })
})
