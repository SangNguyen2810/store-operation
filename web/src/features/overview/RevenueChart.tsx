import type { Filters, RevenuePoint, Store } from '../../api/types'
import { TimeSeriesChart } from '../../components/TimeSeriesChart'
import { formatBucket, formatCurrency } from '../../lib/format'
import { pivotRevenue, totalRevenue } from './pivot'

interface Props {
  points: RevenuePoint[]
  stores: Store[]
  filters: Filters
}

export function RevenueChart({ points, stores, filters }: Props) {
  const { rows, series } = pivotRevenue(points, stores)
  const label = `Revenue per ${filters.granularity}, ${formatBucket(filters.from)} to ${formatBucket(filters.to)}: ${formatCurrency(totalRevenue(points))} in total across ${series.map((s) => s.label).join(' and ')}.`
  return <TimeSeriesChart rows={rows} series={series} ariaLabel={label} formatX={formatBucket} formatY={formatCurrency} />
}
