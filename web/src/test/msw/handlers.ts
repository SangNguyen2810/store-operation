import { http, HttpResponse, delay } from 'msw'
import type { FilmStat, Kpis, Metric, RevenuePoint, Store } from '../../api/types'

/**
 * A small fake of the Go API that answers by the query string, like the real
 * one: figures depend on the store, the revenue series follows the granularity,
 * and the film list follows the store. Paths and JSON field names are the real
 * contract, so a client that drifts from them fails here.
 */

export const STORES: Store[] = [
  { id: 1, label: 'Boksburg, South Africa' },
  { id: 2, label: 'Hamilton, New Zealand' },
]

const metric = (value: number, previous: number): Metric => ({
  value,
  previous,
  deltaPct: previous === 0 ? null : Math.round(((value - previous) / previous) * 10000) / 100,
})

/** Revenue and rentals per store; the rest of the figures are derived from them. */
const FIGURES: Record<string, { revenue: number; rentals: number; customers: number }> = {
  all: { revenue: 300, rentals: 30, customers: 12 },
  '1': { revenue: 200, rentals: 20, customers: 8 },
  '2': { revenue: 100, rentals: 10, customers: 4 },
}

export function kpisFor(store: string | null): Kpis {
  const f = FIGURES[store ?? 'all'] ?? { revenue: 0, rentals: 0, customers: 0 }
  // The previous period is exactly half, so every delta is +100%.
  return {
    revenue: metric(f.revenue, f.revenue / 2),
    rentals: metric(f.rentals, f.rentals / 2),
    activeCustomers: metric(f.customers, f.customers / 2),
    avgPayment: metric(4, 2),
  }
}

const FILMS: Record<string, FilmStat[]> = {
  all: [
    { filmId: 1, title: 'ACADEMY DINOSAUR', rentals: 9, revenue: 90 },
    { filmId: 2, title: 'BUCKET BROTHERHOOD', rentals: 8, revenue: 70 },
  ],
  '1': [{ filmId: 3, title: 'CHAMBER ITALIAN', rentals: 5, revenue: 50 }],
  '2': [{ filmId: 4, title: 'DARKO DORADO', rentals: 4, revenue: 40 }],
}

function series(params: URLSearchParams): RevenuePoint[] {
  const store = params.get('store')
  const ids = store ? [Number(store)] : STORES.map((s) => s.id)
  const bucket = { day: '2022-07-27', week: '2022-07-25', month: '2022-07-01' }[params.get('granularity') ?? 'day'] ?? '2022-07-27'
  return ids.map((storeId) => ({ bucket, storeId, revenue: storeId * 100 }))
}

export const handlers = [
  http.get('*/api/meta', () => HttpResponse.json({ asOf: '2022-07-27' })),
  http.get('*/api/stores', () => HttpResponse.json(STORES)),
  http.get('*/api/kpis', ({ request }) => HttpResponse.json(kpisFor(new URL(request.url).searchParams.get('store')))),
  http.get('*/api/revenue', ({ request }) => HttpResponse.json(series(new URL(request.url).searchParams))),
  http.get('*/api/top/films', ({ request }) =>
    HttpResponse.json(FILMS[new URL(request.url).searchParams.get('store') ?? 'all'] ?? []),
  ),
]

/** A problem response like the Go API's. */
export const problem = (status: number, detail: string) =>
  HttpResponse.json({ title: 'Error', status, detail }, { status, headers: { 'Content-Type': 'application/problem+json' } })

export { http, HttpResponse, delay }
