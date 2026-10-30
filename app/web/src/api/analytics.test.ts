import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchKpis, fetchMeta, fetchRanges, fetchRevenue, fetchStores, fetchTopFilms } from './analytics'
import type { Filters } from './types'

const filters: Filters = { from: '2022-07-01', to: '2022-07-10', store: 2, granularity: 'week' }

function stub() {
  const fn = vi.fn((_url: string, _init?: RequestInit) => Promise.resolve(new Response('[]')))
  vi.stubGlobal('fetch', fn)
  return () => fn.mock.calls.map((c) => c[0])
}

afterEach(() => vi.unstubAllGlobals())

// These URLs are the contract with the Go API.
describe('analytics requests', () => {
  it('meta, ranges and stores take no parameters', async () => {
    const urls = stub()
    await fetchMeta()
    await fetchRanges()
    await fetchStores()
    expect(urls()).toEqual(['/api/meta', '/api/presets', '/api/stores'])
  })

  it('kpis send the range and store but not the granularity', async () => {
    const urls = stub()
    await fetchKpis(filters)
    expect(urls()).toEqual(['/api/kpis?from=2022-07-01&to=2022-07-10&store=2'])
  })

  it('revenue also sends the granularity', async () => {
    const urls = stub()
    await fetchRevenue(filters)
    expect(urls()).toEqual(['/api/revenue?from=2022-07-01&to=2022-07-10&store=2&granularity=week'])
  })

  it('top films default to 10 and accept another limit', async () => {
    const urls = stub()
    await fetchTopFilms(filters)
    await fetchTopFilms({ ...filters, store: null }, 3)
    expect(urls()).toEqual([
      '/api/top/films?from=2022-07-01&to=2022-07-10&store=2&limit=10',
      '/api/top/films?from=2022-07-01&to=2022-07-10&limit=3',
    ])
  })

  it('forwards the abort signal', async () => {
    const fn = vi.fn((_u: string, _i?: RequestInit) => Promise.resolve(new Response('{}')))
    vi.stubGlobal('fetch', fn)
    const ctrl = new AbortController()
    await fetchMeta(ctrl.signal)
    expect(fn.mock.calls[0][1]?.signal).toBe(ctrl.signal)
  })
})
