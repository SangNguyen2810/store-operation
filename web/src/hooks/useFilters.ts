import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { Filters } from '../api/types'
import { parseFilters, serializeFilters } from '../lib/filters'

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
