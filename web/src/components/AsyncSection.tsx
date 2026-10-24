import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'

interface Props<T> {
  title: string
  query: UseQueryResult<T, Error>
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
      <div className="state state-error" role="alert">
        <p>Couldn't load {title.toLowerCase()}: {query.error.message}</p>
        <button type="button" onClick={() => void query.refetch()}>
          Retry
        </button>
      </div>
    )
  } else if (query.isPending) {
    body = (
      <div aria-busy="true" aria-label={`Loading ${title.toLowerCase()}`}>
        {skeleton}
      </div>
    )
  } else if (isEmpty?.(query.data)) {
    body = <p className="state state-empty">No data for this period. Try a wider date range.</p>
  } else {
    body = children(query.data)
  }

  return (
    <section aria-labelledby={id} aria-busy={query.isFetching} className={query.isPlaceholderData ? 'stale' : undefined}>
      <h2 id={id}>{title}</h2>
      {body}
    </section>
  )
}
