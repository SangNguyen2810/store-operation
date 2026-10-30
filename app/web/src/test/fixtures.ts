import type { DateRange, Store } from '../api/types'

/** The ranges the API offers by default, resolved for the as-of date 2022-07-27. The same list the Go API's tests expect. */
export const RANGES: DateRange[] = [
  { id: 'last-7', label: 'Last 7 days', group: 'Recent', from: '2022-07-21', to: '2022-07-27', default: false },
  { id: 'last-14', label: 'Last 14 days', group: 'Recent', from: '2022-07-14', to: '2022-07-27', default: false },
  { id: 'last-30', label: 'Last 30 days', group: 'Recent', from: '2022-06-28', to: '2022-07-27', default: true },
  { id: 'last-60', label: 'Last 60 days', group: 'Recent', from: '2022-05-29', to: '2022-07-27', default: false },
  { id: 'last-90', label: 'Last 90 days', group: 'Recent', from: '2022-04-29', to: '2022-07-27', default: false },
  { id: 'last-180', label: 'Last 180 days', group: 'Recent', from: '2022-01-29', to: '2022-07-27', default: false },
  { id: 'month-to-date', label: 'Month to date', group: 'Periods', from: '2022-07-01', to: '2022-07-27', default: false },
  { id: 'last-month', label: 'Last month', group: 'Periods', from: '2022-06-01', to: '2022-06-30', default: false },
  { id: 'quarter-to-date', label: 'Quarter to date', group: 'Periods', from: '2022-07-01', to: '2022-07-27', default: false },
  { id: 'last-quarter', label: 'Last quarter', group: 'Periods', from: '2022-04-01', to: '2022-06-30', default: false },
  { id: '2022-02', label: 'February 2022', group: 'Fixed', from: '2022-02-01', to: '2022-02-28', default: false },
  { id: '2022-03', label: 'March 2022', group: 'Fixed', from: '2022-03-01', to: '2022-03-31', default: false },
  { id: '2022-04', label: 'April 2022', group: 'Fixed', from: '2022-04-01', to: '2022-04-30', default: false },
  { id: '2022-05', label: 'May 2022', group: 'Fixed', from: '2022-05-01', to: '2022-05-31', default: false },
  { id: '2022-06', label: 'June 2022', group: 'Fixed', from: '2022-06-01', to: '2022-06-30', default: false },
  { id: '2022-q1', label: 'Q1 2022', group: 'Fixed', from: '2022-01-01', to: '2022-03-31', default: false },
  { id: '2022-q2', label: 'Q2 2022', group: 'Fixed', from: '2022-04-01', to: '2022-06-30', default: false },
  { id: 'all-data', label: 'All data', group: 'Fixed', from: '2022-01-23', to: '2022-07-27', default: false },
]

export const ASOF = '2022-07-27'

/** A store with every detail filled in; override what a test cares about. */
export function makeStore(id: number, label: string, overrides: Partial<Store> = {}): Store {
  const [city = '', country = ''] = label.split(', ')
  return {
    id,
    label,
    manager: 'Pat Manager',
    phone: '5550100',
    address: '1 Main Street',
    district: 'Central',
    postalCode: '12345',
    city,
    country,
    inventoryItems: 1000 + id,
    activeCustomers: 100 + id,
    ...overrides,
  }
}

export const STORES: Store[] = [makeStore(1, 'Boksburg, South Africa'), makeStore(2, 'Hamilton, New Zealand')]
