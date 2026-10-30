import type { Filters } from '../api/types'

/** One place for cache keys, so a key is never spelled two ways. */
export const queryKeys = {
  meta: ['meta'] as const,
  ranges: ['ranges'] as const,
  stores: ['stores'] as const,
  kpis: (f: Filters) => ['kpis', f.from, f.to, f.store] as const,
  revenue: (f: Filters) => ['revenue', f.from, f.to, f.store, f.granularity] as const,
  topFilms: (f: Filters) => ['topFilms', f.from, f.to, f.store] as const,
}
