import type { DateRange, Filters, Granularity } from '../api/types'

export const GRANULARITIES: Granularity[] = ['day', 'week', 'month']

/** The ranges that get a button of their own, by the id the API gives them. The rest are in the menu. */
export const QUICK_RANGES = [
  { id: 'last-7', label: '7d' },
  { id: 'last-30', label: '30d' },
  { id: 'last-90', label: '90d' },
] as const

/** The range the API marks as the default. */
export function defaultRange(ranges: DateRange[]): DateRange {
  const range = ranges.find((r) => r.default) ?? ranges[0]
  if (!range) throw new Error('the API offered no date ranges')
  return range
}

export function defaultFilters(ranges: DateRange[]): Filters {
  const { from, to } = defaultRange(ranges)
  return { from, to, store: null, granularity: 'day' }
}

/** The offered range with exactly these days, if any. */
export function findRange(ranges: DateRange[], from: string, to: string): DateRange | undefined {
  return ranges.find((r) => r.from === from && r.to === to)
}

/**
 * How many filters in the collapsible panel differ from the default: a range
 * that has no button of its own, a store, or a non-default grouping.
 */
export function activePanelFilters(f: Filters, ranges: DateRange[]): number {
  const base = defaultFilters(ranges)
  const current = findRange(ranges, f.from, f.to)
  const hasButton = QUICK_RANGES.some((q) => q.id === current?.id)
  const isDefault = f.from === base.from && f.to === base.to
  return (hasButton || isDefault ? 0 : 1) + (f.store !== null ? 1 : 0) + (f.granularity !== base.granularity ? 1 : 0)
}

export function rangeFilters(current: Filters, range: DateRange): Filters {
  return { ...current, from: range.from, to: range.to }
}

/** Reads filters from the URL. Anything missing, or not one of the offered ranges, falls back to the default. */
export function parseFilters(params: URLSearchParams, ranges: DateRange[]): Filters {
  const base = defaultFilters(ranges)

  const to = params.get('to')
  const from = params.get('from')
  const range = from && to ? findRange(ranges, from, to) : undefined

  const storeParam = params.get('store')
  const store = storeParam !== null && /^\d+$/.test(storeParam) ? Number(storeParam) : null

  const g = params.get('granularity') as Granularity | null
  const granularity = g && GRANULARITIES.includes(g) ? g : base.granularity

  return { from: range?.from ?? base.from, to: range?.to ?? base.to, store, granularity }
}

/** Writes only non-default values, so shared links stay short. */
export function serializeFilters(f: Filters, ranges: DateRange[]): URLSearchParams {
  const base = defaultFilters(ranges)
  const params = new URLSearchParams()
  if (f.from !== base.from || f.to !== base.to) {
    params.set('from', f.from)
    params.set('to', f.to)
  }
  if (f.store !== null) params.set('store', String(f.store))
  if (f.granularity !== base.granularity) params.set('granularity', f.granularity)
  return params
}
