// Shared by smoke.js and load.js: where the API is, what to ask it, and what a good answer looks like.
import http from 'k6/http'
import { check } from 'k6'

// Point this at the API, or later at the gateway, with BASE_URL=http://host:port.
export const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '')

// "off" or "on": which cache setup the API under test has. It only labels the results,
// so runs against a cached and an uncached API can be told apart.
export const CACHE = __ENV.CACHE || 'unknown'

// Pagila's payments end on 2022-07-27 (see ASOF_DATE in the API).
export const AS_OF = '2022-07-27'

const ok = http.expectedStatuses({ min: 200, max: 299 })
const okOr = (...codes) => http.expectedStatuses({ min: 200, max: 299 }, ...codes)

export function get(path, { name, expect = ok } = {}) {
  return http.get(`${BASE_URL}${path}`, {
    tags: { name: name || path.split('?')[0] },
    responseCallback: expect,
  })
}

/** A request that should be refused: only this status counts as success. */
export function getExpecting(status, path, name) {
  return get(path, { name, expect: okOr(status) })
}

const isJson = (r) => (r.headers['Content-Type'] || '').startsWith('application/json')
const body = (r) => {
  try {
    return r.json()
  } catch (_) {
    return null
  }
}
const isMetric = (m) => m && typeof m.value === 'number' && typeof m.previous === 'number' && 'deltaPct' in m

// Each checker asserts the status, the content type and the shape the web app relies on.
export const expectOk = {
  health: (r) => check(r, { 'healthz: 200 ok': (r) => r.status === 200 && body(r)?.status === 'ok' }),
  meta: (r) => check(r, { 'meta: asOf is a date': (r) => r.status === 200 && /^\d{4}-\d{2}-\d{2}$/.test(body(r)?.asOf || '') }),
  stores: (r) =>
    check(r, {
      'stores: 200 json': (r) => r.status === 200 && isJson(r),
      'stores: id and label': (r) => Array.isArray(body(r)) && body(r).length > 0 && body(r).every((s) => Number.isInteger(s.id) && s.label),
      'stores: details': (r) =>
        Array.isArray(body(r)) &&
        body(r).every((s) => s.manager && s.address && s.city && s.country && Number.isInteger(s.inventoryItems) && Number.isInteger(s.activeCustomers)),
    }),
  // The offered date ranges: every window the analytics endpoints will accept, with exactly one default.
  presets: (r) =>
    check(r, {
      'presets: 200 json': (r) => r.status === 200 && isJson(r),
      'presets: id, label, group and a window': (r) =>
        Array.isArray(body(r)) &&
        body(r).length > 0 &&
        body(r).every((p) => p.id && p.label && p.group && /^\d{4}-\d{2}-\d{2}$/.test(p.from) && /^\d{4}-\d{2}-\d{2}$/.test(p.to) && p.from <= p.to),
      'presets: exactly one default': (r) => Array.isArray(body(r)) && body(r).filter((p) => p.default).length === 1,
    }),
  kpis: (r) =>
    check(r, {
      'kpis: 200 json': (r) => r.status === 200 && isJson(r),
      'kpis: four metrics': (r) => {
        const k = body(r)
        return k && isMetric(k.revenue) && isMetric(k.rentals) && isMetric(k.activeCustomers) && isMetric(k.avgPayment)
      },
    }),
  revenue: (r) =>
    check(r, {
      'revenue: 200 json': (r) => r.status === 200 && isJson(r),
      'revenue: points have bucket, storeId, revenue': (r) =>
        Array.isArray(body(r)) && body(r).every((p) => /^\d{4}-\d{2}-\d{2}$/.test(p.bucket) && Number.isInteger(p.storeId) && typeof p.revenue === 'number'),
    }),
  // With `spread`, the best film must also earn strictly more than the last one. Lists that end in a
  // tie (many films share the smallest revenue) would otherwise pass even when ranked backwards,
  // so use it on a window wide enough to have a clear winner.
  films: (r, limit, { spread = false } = {}) =>
    check(r, {
      'films: 200 json': (r) => r.status === 200 && isJson(r),
      'films: ranked by revenue, within the limit': (r) => {
        const f = body(r)
        return Array.isArray(f) && f.length <= limit && f.every((x, i) => i === 0 || x.revenue <= f[i - 1].revenue)
      },
      ...(spread && {
        'films: best earns more than the last': (r) => {
          const f = body(r)
          return Array.isArray(f) && f.length === limit && f[0].revenue > f[f.length - 1].revenue
        },
      }),
    }),
  // The router's own 404 and 405 are plain text by design (README: Errors), so only the status is checked there.
  status: (status) => (r) => check(r, { [`${status} status`]: (r) => r.status === status }),
  problem: (status) => (r) =>
    check(r, {
      [`${status} problem+json with a detail`]: (r) =>
        r.status === status && (r.headers['Content-Type'] || '').startsWith('application/problem+json') && typeof body(r)?.title === 'string',
    }),
}
