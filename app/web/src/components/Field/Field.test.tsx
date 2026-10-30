import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Field, FieldLabel } from './Field'

describe('Field', () => {
  it('labels the control it wraps', () => {
    render(
      <Field label="Store">
        <select>
          <option>All</option>
        </select>
      </Field>,
    )
    expect(screen.getByLabelText('Store')).toBeInstanceOf(HTMLSelectElement)
  })

  it('is marked wide only when asked', () => {
    const { container, rerender } = render(<Field label="A"><input /></Field>)
    expect(container.querySelector('label')).not.toHaveAttribute('data-wide')
    rerender(<Field label="A" wide><input /></Field>)
    expect(container.querySelector('label')).toHaveAttribute('data-wide', 'true')
  })
})

describe('FieldLabel', () => {
  it('renders its text and passes native attributes through', () => {
    render(<FieldLabel id="range" className="custom">Range</FieldLabel>)
    expect(screen.getByText('Range')).toHaveAttribute('id', 'range')
    expect(screen.getByText('Range')).toHaveClass('custom')
  })
})
