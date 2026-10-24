import type { Filters, Granularity } from '../../api/types'
import { addDays, isIsoDate } from '../../lib/dates'

export const DEFAULT_DAYS = 30
export const MAX_DAYS = 366
export const GRANULARITIES: Granularity[] = ['day', 'week', 'month']

export const PRESETS = [
  { label: 'Last 7 days', days: 7 },
  { label: 'Last 30 days', days: 30 },
  { label: 'Last 90 days', days: 90 },
] as const

export function defaultFilters(asOf: string): Filters {
  return { from: addDays(asOf, -(DEFAULT_DAYS - 1)), to: asOf, store: null, granularity: 'day' }
}

export function presetFilters(current: Filters, asOf: string, days: number): Filters {
  return { ...current, from: addDays(asOf, -(days - 1)), to: asOf }
}

/** Reads filters from the URL. Anything missing or invalid falls back to the default. */
export function parseFilters(params: URLSearchParams, asOf: string): Filters {
  const base = defaultFilters(asOf)

  const to = params.get('to')
  const from = params.get('from')
  let range = { from: base.from, to: base.to }
  if (isIsoDate(to) && isIsoDate(from)) {
    const spanDays = (Date.parse(to) - Date.parse(from)) / 86_400_000
    if (spanDays >= 0 && spanDays < MAX_DAYS) range = { from, to }
  }

  const storeParam = params.get('store')
  const store = storeParam !== null && /^\d+$/.test(storeParam) ? Number(storeParam) : null

  const g = params.get('granularity') as Granularity | null
  const granularity = g && GRANULARITIES.includes(g) ? g : base.granularity

  return { ...range, store, granularity }
}

/** Writes only non-default values, so shared links stay short. */
export function serializeFilters(f: Filters, asOf: string): URLSearchParams {
  const base = defaultFilters(asOf)
  const params = new URLSearchParams()
  if (f.from !== base.from || f.to !== base.to) {
    params.set('from', f.from)
    params.set('to', f.to)
  }
  if (f.store !== null) params.set('store', String(f.store))
  if (f.granularity !== base.granularity) params.set('granularity', f.granularity)
  return params
}
