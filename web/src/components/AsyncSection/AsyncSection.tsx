import type { ReactNode } from 'react'
import { ErrorState } from '../ErrorState'

/**
 * The part of a query result this component reads. Any react-query result fits,
 * but nothing here imports react-query, so the component stays pure UI.
 */
export interface QueryState<T> {
  data: T | undefined
  error: Error | null
  isError: boolean
  isPending: boolean
  isFetching: boolean
  isPlaceholderData: boolean
  refetch: () => unknown
}

interface Props<T> {
  title: string
  query: QueryState<T>
  isEmpty?: (data: T) => boolean
  skeleton: ReactNode
  children: (data: T) => ReactNode
}

/**
 * One place for the loading / error / empty / ready states of a query-backed section.
 * While new filters load, the previous data stays visible and the section is marked busy.
 */
export function AsyncSection<T>({ title, query, isEmpty, skeleton, children }: Props<T>) {
  const id = `section-${title.toLowerCase().replace(/\W+/g, '-')}`
  let body: ReactNode

  if (query.isError) {
    body = (
      <ErrorState
        message={`Couldn't load ${title.toLowerCase()}: ${query.error?.message ?? 'unknown error'}`}
        onRetry={() => void query.refetch()}
      />
    )
  } else if (query.isPending) {
    body = (
      <div aria-busy="true" aria-label={`Loading ${title.toLowerCase()}`}>
        {skeleton}
      </div>
    )
  } else if (isEmpty?.(query.data as T)) {
    body = <p className="state state-empty">No data for this period. Try a wider date range.</p>
  } else {
    body = children(query.data as T)
  }

  return (
    <section aria-labelledby={id} aria-busy={query.isFetching} className={query.isPlaceholderData ? 'stale' : undefined}>
      <h2 id={id}>{title}</h2>
      {body}
    </section>
  )
}
