import type { Kpis, Metric } from '../../api/types'
import type { KpiTrend } from '../../components/KpiCard'
import type { KpiItem } from '../../components/KpiGrid'
import { formatCurrency, formatDelta, formatInteger } from '../../lib/format'

const CARDS: { key: keyof Kpis; label: string; format: (v: number) => string }[] = [
  { key: 'revenue', label: 'Revenue', format: formatCurrency },
  { key: 'rentals', label: 'Rentals', format: formatInteger },
  { key: 'activeCustomers', label: 'Active customers', format: formatInteger },
  { key: 'avgPayment', label: 'Average payment', format: formatCurrency },
]

/** null when there was no previous value to compare against. */
export function toTrend(metric: Metric): KpiTrend | null {
  if (metric.deltaPct === null) return null
  const pct = metric.deltaPct
  const direction = pct > 0 ? 'up' : pct < 0 ? 'down' : 'flat'
  return {
    direction,
    text: formatDelta(pct),
    spokenLabel: `${direction === 'flat' ? 'unchanged' : direction} ${Math.abs(pct).toFixed(1)}% versus previous period`,
  }
}

export function toKpiItems(kpis: Kpis): KpiItem[] {
  return CARDS.map(({ key, label, format }) => ({
    id: key,
    label,
    value: format(kpis[key].value),
    caption: `prev. ${format(kpis[key].previous)}`,
    trend: toTrend(kpis[key]),
  }))
}
