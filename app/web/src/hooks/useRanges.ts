import { useQuery } from '@tanstack/react-query'
import { fetchRanges } from '../api/analytics'
import { queryKeys } from './queryKeys'

/** The date ranges the API offers. They can change on the server, so they are re-read now and then. */
export const useRanges = () =>
  useQuery({ queryKey: queryKeys.ranges, queryFn: ({ signal }) => fetchRanges(signal), staleTime: 5 * 60_000 })
