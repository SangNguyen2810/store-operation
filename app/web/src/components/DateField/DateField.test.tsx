import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { DateField } from './DateField'

describe('DateField', () => {
  it('is a date input labelled by its text, with min and max', () => {
    render(<DateField label="From" value="2022-06-28" max="2022-07-27" onChange={() => {}} />)
    const input = screen.getByLabelText('From')
    expect(input).toHaveAttribute('type', 'date')
    expect(input).toHaveValue('2022-06-28')
    expect(input).toHaveAttribute('max', '2022-07-27')
  })

  it('reports what the browser gives, including an empty value', () => {
    const onChange = vi.fn()
    render(<DateField label="From" value="2022-06-28" onChange={onChange} />)
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2022-07-01' } })
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '' } })
    expect(onChange).toHaveBeenNthCalledWith(1, '2022-07-01')
    expect(onChange).toHaveBeenNthCalledWith(2, '')
  })
})
