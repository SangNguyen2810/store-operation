export type TrendDirection = 'up' | 'down' | 'flat'

export interface KpiTrend {
  direction: TrendDirection
  /** What is shown, for example "+3.2%". */
  text: string
  /** What a screen reader hears, for example "up 3.2% versus previous period". */
  spokenLabel: string
}

export interface KpiCardProps {
  label: string
  value: string
  /** Small line under the trend, for example "prev. $10,973.36". */
  caption: string
  /** null when there is nothing to compare against. */
  trend: KpiTrend | null
}

const ARROWS: Record<TrendDirection, string> = { up: '▲', down: '▼', flat: '■' }

function Trend({ trend }: { trend: KpiTrend | null }) {
  if (trend === null) {
    return <span className="delta delta-none" aria-label="No previous period to compare">—</span>
  }
  return (
    <span className={`delta delta-${trend.direction}`} aria-label={trend.spokenLabel}>
      <span aria-hidden="true">{ARROWS[trend.direction]} {trend.text}</span>
    </span>
  )
}

/** One figure with its change. Colour is never the only signal: the arrow and text carry it too. */
export function KpiCard({ label, value, caption, trend }: KpiCardProps) {
  return (
    <li className="card">
      <span className="kpi-label">{label}</span>
      <strong className="kpi-value">{value}</strong>
      <Trend trend={trend} />
      <span className="kpi-prev">{caption}</span>
    </li>
  )
}
