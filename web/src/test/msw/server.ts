import { setupServer } from 'msw/node'
import { afterAll, afterEach, beforeAll, beforeEach, vi } from 'vitest'
import { handlers } from './handlers'

export const server = setupServer(...handlers)

/** Every request the app made, as "/api/kpis?from=…", in order. */
export const requests: string[] = []

/**
 * Starts the fake API for one test file. Node's fetch cannot take a relative
 * URL like the browser's, so requests are resolved against the page origin
 * first. Any request without a handler fails the test instead of passing silently.
 */
export function setupFakeApi() {
  beforeAll(() => {
    server.listen({ onUnhandledRequest: 'error' })
    server.events.on('request:start', ({ request }) => {
      const u = new URL(request.url)
      requests.push(u.pathname + u.search)
    })
    // listen() patches fetch, so take it afterwards: the wrapper must call MSW's version.
    const mswFetch = globalThis.fetch
    vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) =>
      mswFetch(typeof input === 'string' ? new URL(input, window.location.origin) : input, init),
    )
  })
  beforeEach(() => {
    requests.length = 0
  })
  afterEach(() => server.resetHandlers())
  afterAll(() => {
    server.close()
    vi.unstubAllGlobals()
  })
}

/** Requests whose path is exactly `path`, e.g. requestsTo('/api/kpis'). */
export const requestsTo = (path: string) => requests.filter((r) => r.split('?')[0] === path)
