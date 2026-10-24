import { KpiCard, KpiCardSkeleton } from '../KpiCard'
import type { KpiCardProps } from '../KpiCard'

export type KpiItem = KpiCardProps & { id: string }

/** Cards two across on small screens and four across from 640px. */
const LIST = 'm-0 grid list-none grid-cols-2 gap-2 p-0 sm:grid-cols-4 sm:gap-3'

export function KpiGrid({ items }: { items: KpiItem[] }) {
  return (
    <ul className={LIST}>
      {items.map(({ id, ...card }) => (
        <KpiCard key={id} {...card} />
      ))}
    </ul>
  )
}

export function KpiGridSkeleton({ count = 4 }: { count?: number }) {
  return (
    <ul className={LIST}>
      {Array.from({ length: count }, (_, i) => (
        <KpiCardSkeleton key={i} />
      ))}
    </ul>
  )
}
