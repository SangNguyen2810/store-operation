import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { fetchRevenue } from '../api/analytics'
import type { Filters } from '../api/types'
import { queryKeys } from './queryKeys'

export const useRevenue = (f: Filters) =>
  useQuery({
    queryKey: queryKeys.revenue(f),
    queryFn: ({ signal }) => fetchRevenue(f, signal),
    placeholderData: keepPreviousData,
  })
