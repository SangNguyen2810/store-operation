import { getJson } from './client'
import type { Filters, FilmStat, Kpis, Meta, RevenuePoint, Store } from './types'

const range = (f: Filters) => ({ from: f.from, to: f.to, store: f.store })

export const fetchMeta = (signal?: AbortSignal) => getJson<Meta>('/api/meta', {}, signal)

export const fetchStores = (signal?: AbortSignal) => getJson<Store[]>('/api/stores', {}, signal)

export const fetchKpis = (f: Filters, signal?: AbortSignal) =>
  getJson<Kpis>('/api/kpis', range(f), signal)

export const fetchRevenue = (f: Filters, signal?: AbortSignal) =>
  getJson<RevenuePoint[]>('/api/revenue', { ...range(f), granularity: f.granularity }, signal)

export const fetchTopFilms = (f: Filters, limit = 10, signal?: AbortSignal) =>
  getJson<FilmStat[]>('/api/top/films', { ...range(f), limit }, signal)
