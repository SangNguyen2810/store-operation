import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, getJson } from './client'

function stubFetch(res: Response | (() => Promise<Response>)) {
  const fn = vi.fn((_url: string, _init?: RequestInit) => (typeof res === 'function' ? res() : Promise.resolve(res)))
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => vi.unstubAllGlobals())

describe('getJson', () => {
  it('returns the parsed body', async () => {
    stubFetch(new Response(JSON.stringify({ a: 1 })))
    expect(await getJson('/api/x')).toEqual({ a: 1 })
  })

  it('requests the bare path when there are no params', async () => {
    const fetch = stubFetch(new Response('{}'))
    await getJson('/api/x')
    expect(fetch.mock.calls[0][0]).toBe('/api/x')
  })

  it('builds the query string, skipping null and undefined and keeping zero', async () => {
    const fetch = stubFetch(new Response('{}'))
    await getJson('/api/x', { a: 'one', b: null, c: undefined, d: 0, e: 5 })
    expect(fetch.mock.calls[0][0]).toBe('/api/x?a=one&d=0&e=5')
  })

  it('encodes values', async () => {
    const fetch = stubFetch(new Response('{}'))
    await getJson('/api/x', { q: 'a b&c' })
    expect(fetch.mock.calls[0][0]).toBe('/api/x?q=a+b%26c')
  })

  it('passes the abort signal to fetch', async () => {
    const fetch = stubFetch(new Response('{}'))
    const ctrl = new AbortController()
    await getJson('/api/x', {}, ctrl.signal)
    expect(fetch.mock.calls[0][1]?.signal).toBe(ctrl.signal)
  })

  it('turns a problem response into an ApiError with the detail', async () => {
    stubFetch(new Response(JSON.stringify({ title: 'Bad Request', detail: 'from: want YYYY-MM-DD' }), { status: 400 }))
    const err = await getJson('/api/x').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ name: 'ApiError', status: 400, message: 'from: want YYYY-MM-DD' })
  })

  it('falls back to the title, then to the status text', async () => {
    stubFetch(new Response(JSON.stringify({ title: 'Service Unavailable' }), { status: 503 }))
    await expect(getJson('/api/x')).rejects.toMatchObject({ status: 503, message: 'Service Unavailable' })

    stubFetch(new Response('<html>oops</html>', { status: 502, statusText: 'Bad Gateway' }))
    await expect(getJson('/api/x')).rejects.toMatchObject({ status: 502, message: 'Bad Gateway' })
  })

  it('lets a network failure through untouched', async () => {
    stubFetch(() => Promise.reject(new TypeError('Failed to fetch')))
    await expect(getJson('/api/x')).rejects.toThrow('Failed to fetch')
  })
})
