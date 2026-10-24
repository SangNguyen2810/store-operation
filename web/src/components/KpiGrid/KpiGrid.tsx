import { KpiCard } from '../KpiCard'
import type { KpiCardProps } from '../KpiCard'
import { Skeleton } from '../Skeleton'

export type KpiItem = KpiCardProps & { id: string }

/** Cards two across on small screens and four across from 640px (see index.css). */
export function KpiGrid({ items }: { items: KpiItem[] }) {
  return (
    <ul className="kpis">
      {items.map(({ id, ...card }) => (
        <KpiCard key={id} {...card} />
      ))}
    </ul>
  )
}

export function KpiGridSkeleton({ count = 4 }: { count?: number }) {
  return (
    <ul className="kpis">
      {Array.from({ length: count }, (_, i) => (
        <li key={i} className="card">
          <Skeleton height={14} width="50%" />
          <Skeleton height={28} width="70%" />
          <Skeleton height={14} width="40%" />
        </li>
      ))}
    </ul>
  )
}
