export type Granularity = 'day' | 'week' | 'month'

export interface Store {
  id: number
  label: string
}

export interface Metric {
  value: number
  previous: number
  deltaPct: number | null
}

export interface Kpis {
  revenue: Metric
  rentals: Metric
  activeCustomers: Metric
  avgPayment: Metric
}

export interface RevenuePoint {
  bucket: string
  storeId: number
  revenue: number
}

export interface FilmStat {
  filmId: number
  title: string
  rentals: number
  revenue: number
}

export interface Meta {
  asOf: string
}

/** The filters every analytics endpoint understands. Dates are YYYY-MM-DD. */
export interface Filters {
  from: string
  to: string
  store: number | null
  granularity: Granularity
}
