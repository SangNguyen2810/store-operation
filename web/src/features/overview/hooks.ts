import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { fetchKpis, fetchMeta, fetchRevenue, fetchStores, fetchTopFilms } from '../../api/analytics'
import type { Filters } from '../../api/types'
import { parseFilters, serializeFilters } from './filters'

export const queryKeys = {
  meta: ['meta'] as const,
  stores: ['stores'] as const,
  kpis: (f: Filters) => ['kpis', f.from, f.to, f.store] as const,
  revenue: (f: Filters) => ['revenue', f.from, f.to, f.store, f.granularity] as const,
  topFilms: (f: Filters) => ['topFilms', f.from, f.to, f.store] as const,
}

// Keep the previous result on screen while new filters load, so the page doesn't flash.
const keep = { placeholderData: keepPreviousData } as const

export const useMeta = () =>
  useQuery({ queryKey: queryKeys.meta, queryFn: ({ signal }) => fetchMeta(signal), staleTime: Infinity })

export const useStores = () =>
  useQuery({ queryKey: queryKeys.stores, queryFn: ({ signal }) => fetchStores(signal), staleTime: Infinity })

export const useKpis = (f: Filters) =>
  useQuery({ queryKey: queryKeys.kpis(f), queryFn: ({ signal }) => fetchKpis(f, signal), ...keep })

export const useRevenue = (f: Filters) =>
  useQuery({ queryKey: queryKeys.revenue(f), queryFn: ({ signal }) => fetchRevenue(f, signal), ...keep })

export const useTopFilms = (f: Filters) =>
  useQuery({ queryKey: queryKeys.topFilms(f), queryFn: ({ signal }) => fetchTopFilms(f, 10, signal), ...keep })

/** Filters live in the URL, so a reload or a shared link restores the same view. */
export function useFilters(asOf: string): [Filters, (next: Filters) => void] {
  const [params, setParams] = useSearchParams()
  const filters = useMemo(() => parseFilters(params, asOf), [params, asOf])
  const setFilters = useCallback(
    (next: Filters) => setParams(serializeFilters(next, asOf), { replace: false }),
    [setParams, asOf],
  )
  return [filters, setFilters]
}
