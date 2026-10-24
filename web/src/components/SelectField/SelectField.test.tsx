import { render, screen } from '@testing-library/react'
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
    expect(container.querySelector('label')).toHaveClass('field-wide')
  })
})
