import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { FilterBar } from './FilterBar'
import { defaultFilters } from './filters'

const ASOF = '2022-07-27'
const stores = [
  { id: 1, label: 'Boksburg, South Africa' },
  { id: 2, label: 'Hamilton, New Zealand' },
]

function setup() {
  const onChange = vi.fn()
  render(<FilterBar filters={defaultFilters(ASOF)} asOf={ASOF} stores={stores} onChange={onChange} />)
  return onChange
}

describe('FilterBar', () => {
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
})
