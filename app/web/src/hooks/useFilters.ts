import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { DateRange, Filters } from '../api/types'
import { parseFilters, serializeFilters } from '../lib/filters'

/** Filters live in the URL, so a reload or a shared link restores the same view. */
export function useFilters(ranges: DateRange[]): [Filters, (next: Filters) => void] {
  const [params, setParams] = useSearchParams()
  const filters = useMemo(() => parseFilters(params, ranges), [params, ranges])
  const setFilters = useCallback(
    (next: Filters) => setParams(serializeFilters(next, ranges), { replace: false }),
    [setParams, ranges],
  )
  return [filters, setFilters]
}
