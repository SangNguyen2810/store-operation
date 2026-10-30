import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { DateRange } from '../../api/types'
import { OverviewFilters } from './OverviewFilters'
import { defaultFilters, rangeFilters } from '../../lib/filters'
import { RANGES, STORES } from '../../test/fixtures'

function setup(filters = defaultFilters(RANGES), ranges: DateRange[] = RANGES) {
  const onChange = vi.fn()
  render(<OverviewFilters filters={filters} ranges={ranges} stores={STORES} onChange={onChange} />)
  return onChange
}

const range = (id: string) => RANGES.find((r) => r.id === id)!

describe('OverviewFilters', () => {
  it('has a button for 7, 30 and 90 days and marks the matching one as pressed', () => {
    setup()
    expect(screen.getAllByRole('button', { pressed: undefined }).filter((b) => b.hasAttribute('aria-pressed'))).toHaveLength(3)
    expect(screen.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Last 7 days' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Last 90 days' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('applies a quick range', async () => {
    const onChange = setup()
    await userEvent.click(screen.getByRole('button', { name: 'Last 7 days' }))
    expect(onChange).toHaveBeenCalledWith({ from: '2022-07-21', to: '2022-07-27', store: null, granularity: 'day' })
  })

  it('only has buttons for quick ranges the API offers', () => {
    setup(defaultFilters(RANGES), RANGES.filter((r) => r.id !== 'last-90'))
    expect(screen.queryByRole('button', { name: 'Last 90 days' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Last 7 days' })).toBeInTheDocument()
  })

  it('lists every offered range in "More ranges", under its group', () => {
    setup()
    const select = screen.getByLabelText('More ranges')
    expect(select).toHaveValue('last-30')
    expect([...select.querySelectorAll('optgroup')].map((g) => g.label)).toEqual(['Recent', 'Periods', 'Fixed'])
    expect(within(select).getAllByRole('option')).toHaveLength(RANGES.length)
    expect(within(screen.getByRole('group', { name: 'Fixed' })).getByRole('option', { name: 'All data' })).toBeInTheDocument()
  })

  it('applies a range chosen from the menu', async () => {
    const onChange = setup()
    await userEvent.selectOptions(screen.getByLabelText('More ranges'), 'Q1 2022')
    expect(onChange).toHaveBeenCalledWith({ from: '2022-01-01', to: '2022-03-31', store: null, granularity: 'day' })
  })

  it('keeps the store and grouping when the range changes', async () => {
    const onChange = setup({ ...defaultFilters(RANGES), store: 2, granularity: 'week' })
    await userEvent.selectOptions(screen.getByLabelText('More ranges'), 'March 2022')
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ from: '2022-03-01', to: '2022-03-31', store: 2, granularity: 'week' }))
  })

  it('shows a menu-only range as selected, with no quick button pressed', () => {
    setup(rangeFilters(defaultFilters(RANGES), range('2022-q1')))
    expect(screen.getByLabelText('More ranges')).toHaveValue('2022-q1')
    for (const name of ['Last 7 days', 'Last 30 days', 'Last 90 days']) {
      expect(screen.getByRole('button', { name })).toHaveAttribute('aria-pressed', 'false')
    }
  })

  it('shows a quick range in the menu too', () => {
    setup(rangeFilters(defaultFilters(RANGES), range('last-7')))
    expect(screen.getByLabelText('More ranges')).toHaveValue('last-7')
    expect(screen.getByRole('button', { name: 'Last 7 days' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('has no free date entry', () => {
    setup()
    expect(screen.queryByLabelText('From')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('To')).not.toBeInTheDocument()
  })

  it('changes store with the keyboard-reachable select', async () => {
    const onChange = setup()
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'Hamilton, New Zealand')
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ store: 2 }))
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'All stores')
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ store: null }))
  })

  it('changes granularity', async () => {
    const onChange = setup()
    await userEvent.selectOptions(screen.getByLabelText('Group by'), 'Week')
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ granularity: 'week' }))
  })

  it('toggles the collapsible panel and reports its state', async () => {
    setup()
    const toggle = screen.getByRole('button', { name: /^Filters/ })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(document.getElementById(toggle.getAttribute('aria-controls')!)).toHaveAttribute('data-open', 'false')
    await userEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(document.getElementById(toggle.getAttribute('aria-controls')!)).toHaveAttribute('data-open', 'true')
  })

  it('shows no badge, and a disabled reset, at the defaults', () => {
    setup()
    expect(screen.queryByLabelText(/active$/)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset filters' })).toBeDisabled()
  })

  it('counts panel filters that differ from the default and resets them', async () => {
    const onChange = setup({ ...defaultFilters(RANGES), store: 2, granularity: 'week' })
    expect(screen.getByLabelText('2 active')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Reset filters' }))
    expect(onChange).toHaveBeenCalledWith(defaultFilters(RANGES))
  })

  it('counts a range that is only in the menu, but not one with a button', () => {
    const { unmount } = render(
      <OverviewFilters filters={rangeFilters(defaultFilters(RANGES), range('last-7'))} ranges={RANGES} stores={STORES} onChange={vi.fn()} />,
    )
    expect(screen.queryByLabelText(/active$/)).not.toBeInTheDocument()
    unmount()
    render(
      <OverviewFilters filters={rangeFilters(defaultFilters(RANGES), range('2022-q1'))} ranges={RANGES} stores={STORES} onChange={vi.fn()} />,
    )
    expect(screen.getByLabelText('1 active')).toBeInTheDocument()
  })
})
