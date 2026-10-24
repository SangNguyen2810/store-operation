import { useQuery } from '@tanstack/react-query'
import { fetchStores } from '../api/analytics'
import { queryKeys } from './queryKeys'

export const useStores = () =>
  useQuery({ queryKey: queryKeys.stores, queryFn: ({ signal }) => fetchStores(signal), staleTime: Infinity })
