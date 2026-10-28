// A realistic mix of dashboard traffic, ramped up to 20 users: `make k6-load`.
// Each virtual user behaves like someone opening the dashboard and then changing filters.
import { sleep } from 'k6'
import { AS_OF, CACHE, expectOk, get } from './lib.js'

export const options = {
  tags: { cache: CACHE },
  stages: [
    { duration: '10s', target: 20 },
    { duration: '30s', target: 20 },
    { duration: '5s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
    http_req_duration: ['p(95)<200'],
    // Each analytics endpoint gets its own limit (this also makes k6 report them separately).
    'http_req_duration{name:/api/kpis}': ['p(95)<250'],
    'http_req_duration{name:/api/revenue}': ['p(95)<250'],
    'http_req_duration{name:/api/top/films}': ['p(95)<250'],
  },
}

// A small, fixed set of choices, as a real dashboard has (presets, two stores, three groupings),
// so repeated requests exist and a cache has something to hit.
const WINDOWS = [
  `from=2022-07-21&to=${AS_OF}`, // last 7 days
  `from=2022-06-28&to=${AS_OF}`, // last 30 days
  `from=2022-04-29&to=${AS_OF}`, // last 90 days
  'from=2022-03-01&to=2022-03-31',
  'from=2022-01-01&to=2022-06-30',
]
const STORES = ['', '', '&store=1', '&store=2'] // "all stores" is the most common
const GRANULARITIES = ['day', 'week', 'month']
const pick = (xs) => xs[Math.floor(Math.random() * xs.length)]

function view() {
  const win = pick(WINDOWS)
  const store = pick(STORES)
  expectOk.kpis(get(`/api/kpis?${win}${store}`))
  expectOk.revenue(get(`/api/revenue?${win}${store}&granularity=${pick(GRANULARITIES)}`))
  expectOk.films(get(`/api/top/films?${win}${store}&limit=10`), 10)
}

export default function () {
  // Opening the dashboard.
  expectOk.meta(get('/api/meta'))
  expectOk.stores(get('/api/stores'))
  view()
  // Then a few filter changes, with a think time between them.
  for (let i = 0; i < 3; i++) {
    sleep(0.5 + Math.random())
    view()
  }
}
