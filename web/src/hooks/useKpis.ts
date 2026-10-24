import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { fetchKpis } from '../api/analytics'
import type { Filters } from '../api/types'
import { queryKeys } from './queryKeys'

// Keeping the previous result on screen while new filters load stops the page flashing.
export const useKpis = (f: Filters) =>
  useQuery({
    queryKey: queryKeys.kpis(f),
    queryFn: ({ signal }) => fetchKpis(f, signal),
    placeholderData: keepPreviousData,
  })
