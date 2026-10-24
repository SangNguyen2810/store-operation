import type { Kpis, Metric } from '../../api/types'
import { Skeleton } from '../../components/Skeleton'
import { formatCurrency, formatDelta, formatInteger } from '../../lib/format'

const CARDS: { key: keyof Kpis; label: string; format: (v: number) => string }[] = [
  { key: 'revenue', label: 'Revenue', format: formatCurrency },
  { key: 'rentals', label: 'Rentals', format: formatInteger },
  { key: 'activeCustomers', label: 'Active customers', format: formatInteger },
  { key: 'avgPayment', label: 'Average payment', format: formatCurrency },
]

function Delta({ metric }: { metric: Metric }) {
  if (metric.deltaPct === null) {
    return <span className="delta delta-none" aria-label="No previous period to compare">—</span>
  }
  const pct = metric.deltaPct
  const dir = pct > 0 ? 'up' : pct < 0 ? 'down' : 'flat'
  const arrow = pct > 0 ? '▲' : pct < 0 ? '▼' : '■'
  return (
    <span className={`delta delta-${dir}`} aria-label={`${dir === 'flat' ? 'unchanged' : dir} ${Math.abs(pct).toFixed(1)}% versus previous period`}>
      <span aria-hidden="true">{arrow} {formatDelta(pct)}</span>
    </span>
  )
}

export function KpiCards({ kpis }: { kpis: Kpis }) {
  return (
    <ul className="kpis">
      {CARDS.map(({ key, label, format }) => (
        <li key={key} className="card">
          <span className="kpi-label">{label}</span>
          <strong className="kpi-value">{format(kpis[key].value)}</strong>
          <Delta metric={kpis[key]} />
          <span className="kpi-prev">prev. {format(kpis[key].previous)}</span>
        </li>
      ))}
    </ul>
  )
}

export function KpiSkeleton() {
  return (
    <ul className="kpis">
      {CARDS.map((c) => (
        <li key={c.key} className="card">
          <Skeleton height={14} width="50%" />
          <Skeleton height={28} width="70%" />
          <Skeleton height={14} width="40%" />
        </li>
      ))}
    </ul>
  )
}
