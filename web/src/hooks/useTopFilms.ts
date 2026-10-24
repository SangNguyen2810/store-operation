import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { fetchTopFilms } from '../api/analytics'
import type { Filters } from '../api/types'
import { queryKeys } from './queryKeys'

const LIMIT = 10

export const useTopFilms = (f: Filters) =>
  useQuery({
    queryKey: queryKeys.topFilms(f),
    queryFn: ({ signal }) => fetchTopFilms(f, LIMIT, signal),
    placeholderData: keepPreviousData,
  })
