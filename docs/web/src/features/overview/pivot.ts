import type { RevenuePoint, Store } from '../../api/types'
import type { Row, Series } from '../../components/TimeSeriesChart'

/** Turns one-row-per-(bucket, store) into one row per bucket with a column per store. */
export function pivotRevenue(points: RevenuePoint[], stores: Store[]) {
  const rows = new Map<string, Row>()
  const ids = new Set<number>()
  for (const p of points) {
    ids.add(p.storeId)
    const row = rows.get(p.bucket) ?? { x: p.bucket }
    row[`s${p.storeId}`] = p.revenue
    rows.set(p.bucket, row)
  }
  const label = (id: number) => stores.find((s) => s.id === id)?.label ?? `Store ${id}`
  const series: Series[] = [...ids].sort((a, b) => a - b).map((id) => ({ key: `s${id}`, label: label(id) }))
  return { rows: [...rows.values()], series }
}

export function totalRevenue(points: RevenuePoint[]): number {
  return points.reduce((sum, p) => sum + p.revenue, 0)
}
