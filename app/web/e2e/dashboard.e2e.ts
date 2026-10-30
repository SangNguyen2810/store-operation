import { expect, test } from '@playwright/test'
import { API, apiKpis, card, chart, dashboardReady, int, keyFigures, revenue, topFilms, usd } from './helpers.ts'

// Real browser, real production build, real API, real Pagila. Nothing is mocked unless a test says so.

test('the overview shows the real figures for the last 30 days', async ({ page, request }) => {
  const expected = await apiKpis(request, '?from=2022-06-28&to=2022-07-27')
  expect(expected.revenue.value).toBeGreaterThan(0) // guards against a vacuous comparison on an empty database

  await page.goto('/')
  await dashboardReady(page)

  await expect(page.getByText('Data as of Jul 27, 2022')).toBeVisible()
  await expect(card(page, 'Revenue')).toContainText(usd(expected.revenue.value))
  await expect(card(page, 'Rentals')).toContainText(int(expected.rentals.value))
  await expect(card(page, 'Active customers')).toContainText(int(expected.activeCustomers.value))
  await expect(card(page, 'Average payment')).toContainText(usd(expected.avgPayment.value))
  await expect(card(page, 'Revenue')).toContainText(`prev. ${usd(expected.revenue.previous)}`)

  await expect(page.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
  await expect(page).toHaveURL(/\/$/)
})

test('the store filter changes the URL and every section, and survives a reload', async ({ page, request }) => {
  const all = await apiKpis(request, '?from=2022-06-28&to=2022-07-27')
  const store1 = await apiKpis(request, '?from=2022-06-28&to=2022-07-27&store=1')
  expect(store1.revenue.value).not.toBe(all.revenue.value)

  await page.goto('/')
  await dashboardReady(page)

  const label = (await page.getByLabel('Store').locator('option').allTextContents()).find((t) => t !== 'All stores')!
  await page.getByLabel('Store').selectOption({ label })

  await expect(page).toHaveURL(/\?store=\d+$/)
  const id = new URL(page.url()).searchParams.get('store')
  const expected = await apiKpis(request, `?from=2022-06-28&to=2022-07-27&store=${id}`)
  await expect(card(page, 'Revenue')).toContainText(usd(expected.revenue.value))
  await expect(card(page, 'Rentals')).toContainText(int(expected.rentals.value))

  await page.reload()
  await dashboardReady(page)
  await expect(page.getByLabel('Store')).toHaveValue(id!)
  await expect(card(page, 'Revenue')).toContainText(usd(expected.revenue.value))
})

test('presets and browser Back move through the filter history', async ({ page, request }) => {
  const week = await apiKpis(request, '?from=2022-07-21&to=2022-07-27')
  await page.goto('/')
  await dashboardReady(page)

  await page.getByRole('button', { name: 'Last 7 days' }).click()
  await expect(page).toHaveURL(/\?from=2022-07-21&to=2022-07-27$/)
  await expect(card(page, 'Revenue')).toContainText(usd(week.revenue.value))

  await page.goBack()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
})

test('"More ranges" offers the API\'s list and a choice reloads the figures', async ({ page, request }) => {
  const presets = (await (await request.get(`${API}/api/presets`)).json()) as { id: string; label: string; from: string; to: string }[]
  const q1 = presets.find((r) => r.id === '2022-q1')!
  const expected = await apiKpis(request, `?from=${q1.from}&to=${q1.to}`)

  await page.goto('/')
  await dashboardReady(page)
  await expect(page.getByLabel('More ranges').locator('option')).toHaveCount(presets.length)

  await page.getByLabel('More ranges').selectOption({ label: q1.label })
  await expect(page).toHaveURL(new RegExp(`\\?from=${q1.from}&to=${q1.to}$`))
  await expect(card(page, 'Revenue')).toContainText(usd(expected.revenue.value))
  await expect(page.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'false')
})

test('the API refuses a window that is not one of the offered ranges, and the app falls back to the default', async ({ page, request }) => {
  const refused = await request.get(`${API}/api/kpis?from=2022-03-05&to=2022-03-20`)
  expect(refused.status()).toBe(400)

  await page.goto('/?from=2022-03-05&to=2022-03-20')
  await dashboardReady(page)
  await expect(page.getByLabel('More ranges')).toHaveValue('last-30')
})

test('a line under the filters describes the stores, and the selected store', async ({ page, request }) => {
  const stores = (await (await request.get(`${API}/api/stores`)).json()) as { id: number; label: string; manager: string; address: string; inventoryItems: number }[]
  const first = stores[0]

  await page.goto('/')
  await dashboardReady(page)
  await expect(page.getByTestId('store-details')).toContainText(`${stores.length} stores`)

  await page.getByLabel('Store').selectOption({ label: first.label })
  await expect(page.getByTestId('store-details')).toContainText(first.manager)
  await expect(page.getByTestId('store-details')).toContainText(first.address)
  await expect(page.getByTestId('store-details')).toContainText(`${first.inventoryItems.toLocaleString('en-US')} inventory items`)
})

test('grouping by month changes the chart data table', async ({ page }) => {
  await page.goto('/')
  await dashboardReady(page)
  await revenue(page).getByText('Show data as a table').click()
  const dailyRows = await revenue(page).getByRole('row').count()

  await page.getByLabel('Group by').selectOption('month')
  await expect(page).toHaveURL(/granularity=month/)
  await expect.poll(() => revenue(page).getByRole('row').count()).toBeLessThan(dailyRows)
})

test('Reset filters returns to the defaults', async ({ page, request }) => {
  const all = await apiKpis(request, '?from=2022-06-28&to=2022-07-27')
  await page.goto('/?from=2022-03-01&to=2022-03-31&granularity=week')
  await dashboardReady(page)
  await expect(page.getByLabel('More ranges')).toHaveValue('2022-03')

  await page.getByRole('button', { name: 'Reset filters' }).click()

  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByLabel('More ranges')).toHaveValue('last-30')
  await expect(page.getByLabel('Group by')).toHaveValue('day')
  await expect(card(page, 'Revenue')).toContainText(usd(all.revenue.value))
  await expect(page.getByRole('button', { name: 'Reset filters' })).toBeDisabled()
})

test('an unknown address shows the 404 page and the link goes back', async ({ page }) => {
  await page.goto('/no/such/page')
  await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible()
  await expect(page).toHaveTitle(/Page not found/)

  await page.getByRole('link', { name: 'Back to the dashboard' }).click()
  await dashboardReady(page)
  await expect(page).toHaveURL(/\/$/)
})

test('an API failure shows an error in that section only, and Retry recovers it', async ({ page }) => {
  // The app retries a failed query once, so it takes two failures to reach the error state.
  let failures = 2
  await page.route('**/api/top/films*', (route) => {
    if (failures-- > 0) {
      return route.fulfill({
        status: 503,
        contentType: 'application/problem+json',
        body: JSON.stringify({ title: 'Service Unavailable', status: 503, detail: 'database unreachable' }),
      })
    }
    return route.continue()
  })

  await page.goto('/')
  const alert = page.getByRole('alert')
  await expect(alert).toContainText("Couldn't load top films: database unreachable")
  await expect(keyFigures(page).getByRole('listitem')).toHaveCount(4) // the other sections still work
  await expect(chart(page)).toBeVisible()

  await alert.getByRole('button', { name: 'Retry' }).click()
  await expect(alert).toHaveCount(0)
  await expect(topFilms(page).getByRole('row').nth(1)).toBeVisible()
})

test('hovering the chart shows a tooltip with each store formatted as money', async ({ page }) => {
  await page.goto('/')
  await dashboardReady(page)
  const box = (await chart(page).boundingBox())!
  await page.mouse.move(box.x + box.width * 0.6, box.y + box.height * 0.4)

  const tooltip = page.locator('.recharts-tooltip-wrapper')
  await expect(tooltip).toBeVisible()
  await expect(tooltip).toContainText(/\$\d/)
  await expect(tooltip).toContainText(/Jun|Jul/)
})

test.describe('on a phone', () => {
  test.use({ viewport: { width: 375, height: 700 }, hasTouch: true })

  test('the filters are folded away until opened, and nothing scrolls sideways', async ({ page }) => {
    await page.goto('/')
    await dashboardReady(page)

    await expect(page.getByLabel('Store')).toBeHidden()
    const toggle = page.getByRole('button', { name: /^Filters/ })
    await expect(toggle).toHaveAttribute('aria-expanded', 'false')

    await toggle.tap()
    await expect(toggle).toHaveAttribute('aria-expanded', 'true')
    await expect(page.getByLabel('Store')).toBeVisible()

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
    expect(overflow).toBeLessThanOrEqual(0)
  })

  test('an active filter is counted on the toggle', async ({ page }) => {
    await page.goto('/?granularity=week')
    await dashboardReady(page)
    await expect(page.getByRole('button', { name: /^Filters/ }).getByLabel('1 active')).toBeVisible()
  })

  test('the films table drops the rentals column and puts the count under the title', async ({ page }) => {
    await page.goto('/')
    await dashboardReady(page)
    await expect(topFilms(page).getByRole('columnheader', { name: 'Rentals' })).toBeHidden()
    await expect(topFilms(page).getByText(/\d+ rentals/).first()).toBeVisible()
  })
})
