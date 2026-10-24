import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { FilterPanel } from './FilterPanel'

function setup(activeCount = 0, onReset = () => {}) {
  render(
    <FilterPanel label="Filters" primary={<button type="button">Primary</button>} activeCount={activeCount} onReset={onReset}>
      <input aria-label="Field" />
    </FilterPanel>,
  )
  return screen.getByRole('button', { name: /^Filters/ })
}

describe('FilterPanel', () => {
  it('shows the primary controls and the fields', () => {
    setup()
    expect(screen.getByRole('form', { name: 'Filters' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Primary' })).toBeInTheDocument()
    expect(screen.getByLabelText('Field')).toBeInTheDocument()
  })

  it('toggles the fields and reports the state to assistive tech', async () => {
    const toggle = setup()
    const fields = document.getElementById(toggle.getAttribute('aria-controls')!)!
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(fields).toHaveAttribute('data-open', 'false')
    expect(fields).toContainElement(screen.getByLabelText('Field'))
    await userEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(fields).toHaveAttribute('data-open', 'true')
  })

  it('shows no count and no reset button when nothing is active', () => {
    setup(0)
    expect(screen.queryByLabelText(/active$/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Reset filters' })).not.toBeInTheDocument()
  })

  it('shows the active count and a working reset button', async () => {
    const onReset = vi.fn()
    setup(2, onReset)
    expect(screen.getByLabelText('2 active')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Reset filters' }))
    expect(onReset).toHaveBeenCalledOnce()
  })

  it('does not submit the form', async () => {
    setup()
    await userEvent.type(screen.getByLabelText('Field'), '{enter}')
    expect(window.location.search).toBe('')
  })
})
