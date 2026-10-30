import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SelectField } from './SelectField'

const options = [
  { value: '', label: 'All stores' },
  { value: '1', label: 'Boksburg' },
]

describe('SelectField', () => {
  it('is a select labelled by its text, showing the current value', () => {
    render(<SelectField label="Store" value="1" options={options} onChange={() => {}} />)
    expect(screen.getByLabelText('Store')).toHaveValue('1')
    expect(screen.getByRole('option', { name: 'All stores' })).toBeInTheDocument()
  })

  it('reports the chosen value, including the empty one', async () => {
    const onChange = vi.fn()
    render(<SelectField label="Store" value="1" options={options} onChange={onChange} />)
    await userEvent.selectOptions(screen.getByLabelText('Store'), 'All stores')
    expect(onChange).toHaveBeenCalledWith('')
  })

  it('can span the full row', () => {
    const { container } = render(<SelectField label="Store" value="" options={options} onChange={() => {}} wide />)
    expect(container.querySelector('label')).toHaveAttribute('data-wide', 'true')
  })

  it('lists options under a heading when they have a group, and plain ones without', () => {
    render(
      <SelectField
        label="Range"
        value="a"
        onChange={() => {}}
        options={[
          { value: 'x', label: 'Plain' },
          { value: 'a', label: 'A', group: 'Recent' },
          { value: 'b', label: 'B', group: 'Recent' },
          { value: 'c', label: 'C', group: 'Fixed' },
        ]}
      />,
    )
    const recent = screen.getByRole('group', { name: 'Recent' })
    expect(within(recent).getAllByRole('option').map((o) => o.textContent)).toEqual(['A', 'B'])
    expect(within(screen.getByRole('group', { name: 'Fixed' })).getByRole('option', { name: 'C' })).toBeInTheDocument()
    expect(screen.getAllByRole('group')).toHaveLength(2)
    expect(screen.getByRole('option', { name: 'Plain' }).closest('optgroup')).toBeNull()
  })

  it('starts a new heading when a group comes back after another', () => {
    render(
      <SelectField
        label="Range"
        value="a"
        onChange={() => {}}
        options={[
          { value: 'a', label: 'A', group: 'G1' },
          { value: 'b', label: 'B', group: 'G2' },
          { value: 'c', label: 'C', group: 'G1' },
        ]}
      />,
    )
    expect(screen.getAllByRole('group').map((g) => g.getAttribute('label'))).toEqual(['G1', 'G2', 'G1'])
  })
})
