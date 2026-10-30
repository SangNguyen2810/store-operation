import { Skeleton } from '../Skeleton'

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
const TONES: Record<TrendDirection, string> = { up: 'text-up', down: 'text-down', flat: 'text-muted' }

export const CARD = 'grid min-w-0 gap-1 rounded-lg border border-line px-3 py-2.5'
const TREND = 'text-[0.85rem] font-semibold'

function Trend({ trend }: { trend: KpiTrend | null }) {
  if (trend === null) {
    return (
      <span className={`${TREND} text-muted`} data-trend="none" aria-label="No previous period to compare">
        —
      </span>
    )
  }
  return (
    <span className={`${TREND} ${TONES[trend.direction]}`} data-trend={trend.direction} aria-label={trend.spokenLabel}>
      <span aria-hidden="true">{ARROWS[trend.direction]} {trend.text}</span>
    </span>
  )
}

/** One figure with its change. Colour is never the only signal: the arrow and text carry it too. */
export function KpiCard({ label, value, caption, trend }: KpiCardProps) {
  return (
    <li className={CARD}>
      <span className="text-[0.8rem] text-muted">{label}</span>
      <strong className="text-[clamp(1.15rem,5.2vw,1.5rem)]">{value}</strong>
      <Trend trend={trend} />
      <span className="text-[0.8rem] text-muted">{caption}</span>
    </li>
  )
}

export function KpiCardSkeleton() {
  return (
    <li className={CARD}>
      <Skeleton height={14} width="50%" />
      <Skeleton height={28} width="70%" />
      <Skeleton height={14} width="40%" />
    </li>
  )
}
