import { expect, type APIRequestContext, type Page } from '@playwright/test'

/** The gateway the app talks to through the preview proxy; specs read it for the expected figures. */
export const API = 'http://localhost:8180'

interface Metric {
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

export async function apiKpis(request: APIRequestContext, query = ''): Promise<Kpis> {
  const res = await request.get(`${API}/api/kpis${query}`)
  expect(res.ok()).toBe(true)
  return (await res.json()) as Kpis
}

/** Formats like the app does: en-US dollars with cents. */
export const usd = (v: number) => v.toLocaleString('en-US', { style: 'currency', currency: 'USD' })
export const int = (v: number) => v.toLocaleString('en-US')

export const keyFigures = (page: Page) => page.getByRole('region', { name: 'Key figures' })
export const topFilms = (page: Page) => page.getByRole('region', { name: 'Top films' })
export const revenue = (page: Page) => page.getByRole('region', { name: 'Revenue' })
/** The chart itself; the legend icons are images too, so it is found by its description. */
export const chart = (page: Page) => page.getByRole('img', { name: /^Revenue per / })

/** The card for one figure, found by its label. */
export const card = (page: Page, label: string) =>
  keyFigures(page).getByRole('listitem').filter({ has: page.getByText(label, { exact: true }) })

/** Waits until the dashboard has drawn real data in all three sections. */
export async function dashboardReady(page: Page) {
  await expect(page.getByRole('heading', { name: 'Store Operations' })).toBeVisible()
  await expect(keyFigures(page).getByRole('listitem')).toHaveCount(4)
  await expect(topFilms(page).getByRole('row').nth(1)).toBeVisible()
  await expect(chart(page)).toBeVisible()
}
