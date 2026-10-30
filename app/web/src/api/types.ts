export type Granularity = 'day' | 'week' | 'month'

export interface Store {
  id: number
  /** "City, Country" */
  label: string
  manager: string
  phone: string
  address: string
  district: string
  postalCode: string
  city: string
  country: string
  /** Items in this store's inventory. */
  inventoryItems: number
  activeCustomers: number
}

/** A date range the API offers. Dates are YYYY-MM-DD, both ends inclusive. */
export interface DateRange {
  id: string
  label: string
  /** The menu heading it is listed under, for example "Recent". */
  group: string
  from: string
  to: string
  default: boolean
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
