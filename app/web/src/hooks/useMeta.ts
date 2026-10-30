import { useQuery } from '@tanstack/react-query'
import { fetchMeta } from '../api/analytics'
import { queryKeys } from './queryKeys'

/** What "today" is for the dashboard. It never changes during a session. */
export const useMeta = () =>
  useQuery({ queryKey: queryKeys.meta, queryFn: ({ signal }) => fetchMeta(signal), staleTime: Infinity })
