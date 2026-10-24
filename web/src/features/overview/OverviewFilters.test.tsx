import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { OverviewFilters } from './OverviewFilters'
import { defaultFilters } from '../../lib/filters'

const ASOF = '2022-07-27'
const stores = [
  { id: 1, label: 'Boksburg, South Africa' },
  { id: 2, label: 'Hamilton, New Zealand' },
]

function setup(filters = defaultFilters(ASOF)) {
  const onChange = vi.fn()
  render(<OverviewFilters filters={filters} asOf={ASOF} stores={stores} onChange={onChange} />)
  return onChange
}

describe('OverviewFilters', () => {
  it('marks the matching preset as pressed', () => {
    setup()
    expect(screen.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Last 7 days' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('applies a preset', async () => {
    const onChange = setup()
    await userEvent.click(screen.getByRole('button', { name: 'Last 7 days' }))
    expect(onChange).toHaveBeenCalledWith({ from: '2022-07-21', to: ASOF, store: null, granularity: 'day' })
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

  it('accepts a valid date and ignores an empty or reversed one', () => {
    const onChange = setup()
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2022-07-01' } })
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ from: '2022-07-01' }))
    onChange.mockClear()
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2022-08-01' } })
    expect(onChange).not.toHaveBeenCalled()
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
    const onChange = setup({ ...defaultFilters(ASOF), store: 2, granularity: 'week' })
    expect(screen.getByLabelText('2 active')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Reset filters' }))
    expect(onChange).toHaveBeenCalledWith(defaultFilters(ASOF))
  })

  it('counts a custom date range but not a preset one', () => {
    const { unmount } = render(
      <OverviewFilters filters={{ ...defaultFilters(ASOF), from: '2022-07-21' }} asOf={ASOF} stores={stores} onChange={vi.fn()} />,
    )
    expect(screen.queryByLabelText(/active$/)).not.toBeInTheDocument() // 7 days is a preset
    unmount()
    render(
      <OverviewFilters filters={{ ...defaultFilters(ASOF), from: '2022-07-01' }} asOf={ASOF} stores={stores} onChange={vi.fn()} />,
    )
    expect(screen.getByLabelText('1 active')).toBeInTheDocument()
  })
})
