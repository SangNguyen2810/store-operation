// One user, one pass over every endpoint, with the answers checked.
// Fast enough to run on every push: `make k6-smoke`.
import { AS_OF, BASE_URL, CACHE, expectOk, get, getExpecting } from './lib.js'

export const options = {
  vus: 1,
  iterations: 1,
  tags: { cache: CACHE },
  thresholds: {
    checks: ['rate==1'], // every assertion must hold
    http_req_failed: ['rate==0'], // no unexpected status
    http_req_duration: ['p(95)<500'],
  },
}

const window30 = `from=2022-06-28&to=${AS_OF}`

export default function () {
  expectOk.health(get('/healthz'))
  expectOk.meta(get('/api/meta'))
  expectOk.stores(get('/api/stores'))
  expectOk.presets(get('/api/presets'))

  expectOk.kpis(get(`/api/kpis?${window30}`))
  expectOk.kpis(get('/api/kpis')) // no parameters: the default window
  expectOk.kpis(get(`/api/kpis?${window30}&store=1`))
  expectOk.kpis(get(`/api/kpis?${window30}&store=999`)) // unknown store: zeros, not an error

  for (const g of ['day', 'week', 'month']) {
    expectOk.revenue(get(`/api/revenue?${window30}&granularity=${g}`))
  }
  expectOk.films(get(`/api/top/films?${window30}&limit=5`), 5)
  expectOk.films(get('/api/top/films'), 10)
  // Wide window (the "All data" range) with a clear winner, so a list ranked the wrong way round is noticed.
  expectOk.films(get('/api/top/films?from=2022-01-23&to=2022-07-27&limit=5'), 5, { spread: true })

  // Bad input is refused with a problem body, not a 500.
  expectOk.problem(400)(getExpecting(400, '/api/kpis?from=nope', '/api/kpis (bad date)'))
  expectOk.problem(400)(getExpecting(400, '/api/kpis?from=2022-07-02&to=2022-07-01', '/api/kpis (reversed)'))
  // Only the ranges from /api/presets are accepted: any other window, or half a window, is refused.
  expectOk.problem(400)(getExpecting(400, '/api/kpis?from=2022-03-05&to=2022-03-20', '/api/kpis (not an offered range)'))
  expectOk.problem(400)(getExpecting(400, '/api/kpis?from=2022-06-29&to=2022-07-27', '/api/kpis (off by one day)'))
  expectOk.problem(400)(getExpecting(400, '/api/kpis?from=2022-06-28', '/api/kpis (only from)'))
  expectOk.problem(400)(getExpecting(400, '/api/revenue?granularity=year', '/api/revenue (bad granularity)'))
  expectOk.problem(400)(getExpecting(400, '/api/top/films?limit=999', '/api/top/films (bad limit)'))
  expectOk.status(404)(getExpecting(404, '/api/nope', '/api/nope'))
}

export function setup() {
  const r = get('/healthz')
  if (r.status !== 200) {
    throw new Error(`nothing healthy at ${BASE_URL} (status ${r.status}); start the API or set BASE_URL`)
  }
}
