import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test/utils'
import { OverviewPage } from './OverviewPage'

const kpis = {
  revenue: { value: 100, previous: 50, deltaPct: 100 },
  rentals: { value: 10, previous: 5, deltaPct: 100 },
  activeCustomers: { value: 8, previous: 4, deltaPct: 100 },
  avgPayment: { value: 4, previous: 4, deltaPct: 0 },
}

function mockApi(overrides: Record<string, () => Response> = {}) {
  const calls: string[] = []
  const json = (body: unknown) => () => new Response(JSON.stringify(body), { status: 200 })
  const routes: Record<string, () => Response> = {
    '/api/meta': json({ asOf: '2022-07-27' }),
    '/api/stores': json([{ id: 1, label: 'Boksburg, South Africa' }]),
    '/api/kpis': json(kpis),
    '/api/revenue': json([{ bucket: '2022-07-27', storeId: 1, revenue: 100 }]),
    '/api/top/films': json([{ filmId: 1, title: 'ACADEMY DINOSAUR', rentals: 3, revenue: 12.5 }]),
    ...overrides,
  }
  vi.stubGlobal('fetch', (input: string) => {
    calls.push(input)
    const handler = routes[input.split('?')[0]]
    return Promise.resolve(handler ? handler() : new Response('{}', { status: 404 }))
  })
  return calls
}

const kpiRevenue = async () => within(await screen.findByRole('region', { name: 'Key figures' })).findByText('$100.00')

afterEach(() => vi.unstubAllGlobals())

describe('OverviewPage', () => {
  it('loads and renders all sections with default filters', async () => {
    const calls = mockApi()
    renderWithProviders(<OverviewPage />)

    expect(await kpiRevenue()).toBeInTheDocument()
    expect(await screen.findByText('ACADEMY DINOSAUR')).toBeInTheDocument()
    expect(await screen.findByText(/show data as a table/i)).toBeInTheDocument()
    expect(calls).toContain('/api/kpis?from=2022-06-28&to=2022-07-27')
  })

  it('restores filters from the URL', async () => {
    const calls = mockApi()
    renderWithProviders(<OverviewPage />, '/?from=2022-03-01&to=2022-03-31&store=1&granularity=week')

    await kpiRevenue()
    expect(screen.getByLabelText('From')).toHaveValue('2022-03-01')
    expect(screen.getByLabelText('Store')).toHaveValue('1')
    expect(calls).toContain('/api/revenue?from=2022-03-01&to=2022-03-31&store=1&granularity=week')
  })

  it('refetches when a filter changes', async () => {
    const calls = mockApi()
    renderWithProviders(<OverviewPage />)
    await kpiRevenue()

    await userEvent.click(screen.getByRole('button', { name: 'Last 7 days' }))
    await screen.findByRole('button', { name: 'Last 7 days', pressed: true })
    expect(calls).toContain('/api/kpis?from=2022-07-21&to=2022-07-27')
  })

  it('shows the empty state when there is no revenue', async () => {
    mockApi({
      '/api/revenue': () => new Response(JSON.stringify([{ bucket: '2022-07-27', storeId: 1, revenue: 0 }])),
      '/api/top/films': () => new Response('[]'),
    })
    renderWithProviders(<OverviewPage />)
    expect(await screen.findAllByText(/no data for this period/i)).toHaveLength(2)
  })

  it('shows a per-section error and keeps the other sections working', async () => {
    mockApi({
      '/api/top/films': () => new Response(JSON.stringify({ detail: 'database unreachable' }), { status: 500 }),
    })
    renderWithProviders(<OverviewPage />)
    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText(/database unreachable/)).toBeInTheDocument()
    expect(await kpiRevenue()).toBeInTheDocument()
  })

  it('shows a retryable error when the API is unreachable', async () => {
    mockApi({ '/api/meta': () => new Response(JSON.stringify({ detail: 'down' }), { status: 503 }) })
    renderWithProviders(<OverviewPage />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/couldn't reach the api/i)
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})
