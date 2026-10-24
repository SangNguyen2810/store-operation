import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SegmentedControl } from './SegmentedControl'

const options = [
  { value: '7', label: '7d', ariaLabel: 'Last 7 days' },
  { value: '30', label: '30d', ariaLabel: 'Last 30 days' },
]

describe('SegmentedControl', () => {
  it('is a labelled group of toggle buttons with the selected one pressed', () => {
    render(<SegmentedControl label="Range" options={options} value="30" onChange={() => {}} />)
    expect(screen.getByRole('group', { name: 'Range' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Last 7 days' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('shows the short label but exposes the full one', () => {
    render(<SegmentedControl label="Range" options={options} value={null} onChange={() => {}} />)
    expect(screen.getByRole('button', { name: 'Last 7 days' })).toHaveTextContent('7d')
  })

  it('presses nothing when value is null', () => {
    render(<SegmentedControl label="Range" options={options} value={null} onChange={() => {}} />)
    for (const b of screen.getAllByRole('button')) expect(b).toHaveAttribute('aria-pressed', 'false')
  })

  it('reports the chosen value', async () => {
    const onChange = vi.fn()
    render(<SegmentedControl label="Range" options={options} value="30" onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Last 7 days' }))
    expect(onChange).toHaveBeenCalledWith('7')
  })

  it('falls back to the label as the accessible name', () => {
    render(<SegmentedControl label="Range" options={[{ value: 'a', label: 'Alpha' }]} value={null} onChange={() => {}} />)
    expect(screen.getByRole('button', { name: 'Alpha' })).toBeInTheDocument()
  })
})
