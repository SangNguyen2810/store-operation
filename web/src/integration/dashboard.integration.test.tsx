import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { delay, http, HttpResponse, kpisFor, problem } from '../test/msw/handlers'
import { requestsTo, server, setupFakeApi } from '../test/msw/server'
import { renderApp } from '../test/renderApp'

// The real pages, hooks, router and fetch layer, against a fake of the Go API at the network level.
setupFakeApi()

const url = () => screen.getByTestId('location').textContent
const keyFigures = () => screen.findByRole('region', { name: 'Key figures' })
const topFilms = () => screen.findByRole('region', { name: 'Top films' })

async function loaded(initial = '/') {
  renderApp(initial)
  const figures = await keyFigures()
  await within(figures).findByText('$300.00')
  return figures
}

describe('first load', () => {
  it('shows every section from the API with the default 30 days', async () => {
    renderApp()
    const figures = await keyFigures()
    expect(await within(figures).findByText('$300.00')).toBeInTheDocument()
    expect(within(figures).getByText('30')).toBeInTheDocument()
    expect(await screen.findByText('ACADEMY DINOSAUR')).toBeInTheDocument()
    expect(await screen.findByText(/show data as a table/i)).toBeInTheDocument()
    expect(screen.getByText(/data as of jul 27, 2022/i)).toBeInTheDocument()

    expect(requestsTo('/api/kpis')).toEqual(['/api/kpis?from=2022-06-28&to=2022-07-27'])
    expect(requestsTo('/api/revenue')).toEqual(['/api/revenue?from=2022-06-28&to=2022-07-27&granularity=day'])
    expect(requestsTo('/api/top/films')).toEqual(['/api/top/films?from=2022-06-28&to=2022-07-27&limit=10'])
    expect(url()).toBe('/')
  })

  it('names the stores in the chart table once the stores load', async () => {
    renderApp()
    expect(await screen.findByRole('columnheader', { name: 'Boksburg, South Africa' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Hamilton, New Zealand' })).toBeInTheDocument()
  })
})

describe('changing a filter', () => {
  it('store: updates the URL, refetches every section with the store, and shows its figures', async () => {
    await loaded()
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Hamilton, New Zealand')

    expect(url()).toBe('/?store=2')
    expect(await within(await keyFigures()).findByText('$100.00')).toBeInTheDocument()
    expect(await screen.findByText('DARKO DORADO')).toBeInTheDocument()
    expect(screen.queryByText('ACADEMY DINOSAUR')).not.toBeInTheDocument()
    expect(requestsTo('/api/kpis').at(-1)).toBe('/api/kpis?from=2022-06-28&to=2022-07-27&store=2')
    expect(requestsTo('/api/revenue').at(-1)).toBe('/api/revenue?from=2022-06-28&to=2022-07-27&store=2&granularity=day')
    expect(requestsTo('/api/top/films').at(-1)).toBe('/api/top/films?from=2022-06-28&to=2022-07-27&store=2&limit=10')
  })

  it('choosing All stores again removes the store from the URL', async () => {
    await loaded()
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Hamilton, New Zealand')
    await waitFor(() => expect(url()).toBe('/?store=2'))
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'All stores')
    expect(url()).toBe('/')
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
  })

  it('preset: sets the range in the URL and requests it; the default preset keeps the URL clean', async () => {
    await loaded()
    await userEvent.click(screen.getByRole('button', { name: 'Last 7 days' }))
    expect(url()).toBe('/?from=2022-07-21&to=2022-07-27')
    await waitFor(() => expect(requestsTo('/api/kpis').at(-1)).toBe('/api/kpis?from=2022-07-21&to=2022-07-27'))

    await userEvent.click(screen.getByRole('button', { name: 'Last 30 days' }))
    expect(url()).toBe('/')
    expect(screen.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('granularity: refetches only the revenue series', async () => {
    await loaded()
    await screen.findByText(/show data as a table/i)
    const kpiCalls = requestsTo('/api/kpis').length
    const filmCalls = requestsTo('/api/top/films').length

    await userEvent.selectOptions(screen.getByLabelText('Group by'), 'Week')

    expect(url()).toBe('/?granularity=week')
    await waitFor(() => expect(requestsTo('/api/revenue').at(-1)).toContain('granularity=week'))
    expect(requestsTo('/api/kpis')).toHaveLength(kpiCalls)
    expect(requestsTo('/api/top/films')).toHaveLength(filmCalls)
  })

  it('a custom From date is written to the URL together with the To date', async () => {
    await loaded()
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2022-07-01' } })
    await waitFor(() => expect(url()).toBe('/?from=2022-07-01&to=2022-07-27'))
    await waitFor(() => expect(requestsTo('/api/kpis').at(-1)).toBe('/api/kpis?from=2022-07-01&to=2022-07-27'))
  })

  it('a reversed range is ignored and nothing is requested for it', async () => {
    await loaded()
    const before = requestsTo('/api/kpis').length
    fireEvent.change(screen.getByLabelText('To'), { target: { value: '2022-06-01' } }) // earlier than From (2022-06-28)
    expect(url()).toBe('/')
    expect(requestsTo('/api/kpis')).toHaveLength(before)
  })
})

describe('the URL is the state', () => {
  it('restores every filter from a shared link and requests exactly that', async () => {
    renderApp('/?from=2022-03-01&to=2022-03-31&store=1&granularity=week')
    await keyFigures()
    await waitFor(() => expect(screen.getByLabelText('Store')).toHaveValue('1'))
    expect(screen.getByLabelText('From')).toHaveValue('2022-03-01')
    expect(screen.getByLabelText('To')).toHaveValue('2022-03-31')
    expect(screen.getByLabelText('Group by')).toHaveValue('week')
    expect(await within(await keyFigures()).findByText('$200.00')).toBeInTheDocument()
    expect(requestsTo('/api/revenue')).toEqual(['/api/revenue?from=2022-03-01&to=2022-03-31&store=1&granularity=week'])
  })

  it.each([
    ['garbage dates', '/?from=nope&to=2022-13-45'],
    ['a reversed range', '/?from=2022-07-10&to=2022-07-01'],
    ['a range over a year', '/?from=2020-01-01&to=2022-07-27'],
    ['a bad granularity', '/?granularity=year'],
    ['a non-numeric store', '/?store=abc'],
  ])('falls back to the defaults for %s', async (_name, link) => {
    renderApp(link)
    await keyFigures()
    await waitFor(() => expect(requestsTo('/api/kpis')).toHaveLength(1))
    expect(requestsTo('/api/kpis')[0]).toBe('/api/kpis?from=2022-06-28&to=2022-07-27')
    expect(requestsTo('/api/revenue')[0]).toContain('granularity=day')
  })
})

describe('browser history', () => {
  it('Back undoes the last filter change, in the URL, the controls and the data', async () => {
    await loaded()
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Hamilton, New Zealand')
    await userEvent.click(screen.getByRole('button', { name: 'Last 7 days' }))
    await waitFor(() => expect(url()).toBe('/?from=2022-07-21&to=2022-07-27&store=2'))

    await userEvent.click(screen.getByRole('button', { name: 'Browser back' }))
    expect(url()).toBe('/?store=2')
    expect(screen.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')

    await userEvent.click(screen.getByRole('button', { name: 'Browser back' }))
    expect(url()).toBe('/')
    expect(screen.getByLabelText('Store')).toHaveValue('')
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
  })
})

describe('reset', () => {
  it('is disabled with default filters', async () => {
    await loaded()
    expect(screen.getByRole('button', { name: 'Reset filters' })).toBeDisabled()
  })

  it('puts every filter back to the default, clears the URL and refetches', async () => {
    renderApp('/?from=2022-03-01&to=2022-03-31&store=1&granularity=week')
    await waitFor(() => expect(screen.getByLabelText('Store')).toHaveValue('1'))
    await keyFigures()

    await userEvent.click(screen.getByRole('button', { name: 'Reset filters' }))

    expect(url()).toBe('/')
    expect(screen.getByLabelText('Store')).toHaveValue('')
    expect(screen.getByLabelText('From')).toHaveValue('2022-06-28')
    expect(screen.getByLabelText('Group by')).toHaveValue('day')
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
    expect(requestsTo('/api/kpis').at(-1)).toBe('/api/kpis?from=2022-06-28&to=2022-07-27')
    expect(screen.getByRole('button', { name: 'Reset filters' })).toBeDisabled()
  })
})

describe('errors and retry', () => {
  it('a failing section shows the API detail, the other sections keep working, and Retry recovers it', async () => {
    server.use(http.get('*/api/top/films', () => problem(503, 'database unreachable'), { once: true }))
    renderApp()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent("Couldn't load top films: database unreachable")
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
    expect(within(await topFilms()).queryByText('ACADEMY DINOSAUR')).not.toBeInTheDocument()

    await userEvent.click(within(alert).getByRole('button', { name: 'Retry' }))

    expect(await screen.findByText('ACADEMY DINOSAUR')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(requestsTo('/api/top/films')).toHaveLength(2)
  })

  it('a failing KPI section does not stop the chart or the films', async () => {
    server.use(http.get('*/api/kpis', () => problem(500, 'internal error'), { once: true }))
    renderApp()
    expect(await screen.findByRole('alert')).toHaveTextContent(/couldn't load key figures: internal error/i)
    expect(await screen.findByText('ACADEMY DINOSAUR')).toBeInTheDocument()
    expect(await screen.findByText(/show data as a table/i)).toBeInTheDocument()
  })

  it('an unreachable API shows a page-level error and Retry brings the dashboard up', async () => {
    server.use(http.get('*/api/meta', () => HttpResponse.error(), { once: true }))
    renderApp()
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(/couldn't reach the api/i)
    expect(screen.queryByLabelText('Store')).not.toBeInTheDocument()

    await userEvent.click(within(alert).getByRole('button', { name: 'Retry' }))
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
    expect(screen.getByLabelText('Store')).toBeInTheDocument()
  })

  it('a filter change that fails shows the error, and changing the filter back recovers', async () => {
    await loaded()
    server.use(http.get('*/api/kpis', () => problem(500, 'internal error'), { once: true }))
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Boksburg, South Africa')
    expect(await screen.findByRole('alert')).toHaveTextContent(/internal error/)
    expect(url()).toBe('/?store=1')

    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await within(await keyFigures()).findByText('$200.00')).toBeInTheDocument()
  })
})

describe('empty and loading states', () => {
  it('a period without sales says so in both sections', async () => {
    server.use(
      http.get('*/api/revenue', () => HttpResponse.json([{ bucket: '2022-07-27', storeId: 1, revenue: 0 }])),
      http.get('*/api/top/films', () => HttpResponse.json([])),
    )
    renderApp()
    expect(await screen.findAllByText(/no data for this period/i)).toHaveLength(2)
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
  })

  it('keeps the previous figures, marked busy, while new ones load', async () => {
    await loaded()
    const figures = await keyFigures()
    await within(figures).findByText('$300.00')
    server.use(
      http.get('*/api/kpis', async ({ request }) => {
        await delay(150)
        return HttpResponse.json(kpisFor(new URL(request.url).searchParams.get('store')))
      }),
    )
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Boksburg, South Africa')

    await waitFor(() => expect(figures).toHaveAttribute('aria-busy', 'true'))
    expect(within(figures).getByText('$300.00')).toBeInTheDocument()
    expect(await within(figures).findByText('$200.00')).toBeInTheDocument()
    await waitFor(() => expect(figures).toHaveAttribute('aria-busy', 'false'))
  })

  it('when filters change quickly, the last choice wins even if an earlier answer arrives later', async () => {
    await loaded()
    server.use(
      http.get('*/api/kpis', async ({ request }) => {
        const store = new URL(request.url).searchParams.get('store')
        if (store === '1') await delay(300) // the first choice is the slow one
        return HttpResponse.json(kpisFor(store))
      }),
    )
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Boksburg, South Africa')
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Hamilton, New Zealand')

    const figures = await keyFigures()
    expect(await within(figures).findByText('$100.00')).toBeInTheDocument()
    await delay(450)
    expect(within(figures).getByText('$100.00')).toBeInTheDocument()
    expect(within(figures).queryByText('$200.00')).not.toBeInTheDocument()
    expect(url()).toBe('/?store=2')
  })
})

describe('routing', () => {
  it('an unknown address shows the 404 page without calling the API, and the link returns to the dashboard', async () => {
    renderApp('/does/not/exist')
    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
    expect(requestsTo('/api/meta')).toHaveLength(0)

    await userEvent.click(screen.getByRole('link', { name: 'Back to the dashboard' }))
    expect(url()).toBe('/')
    expect(await within(await keyFigures()).findByText('$300.00')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Page not found' })).not.toBeInTheDocument()
  })
})
