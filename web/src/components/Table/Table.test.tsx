import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Table, Td, Th } from './Table'

function setup(extra: { numeric?: boolean } = {}) {
  render(
    <Table aria-label="Figures">
      <thead>
        <tr>
          <Th scope="col">Name</Th>
          <Th scope="col" {...extra}>Total</Th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <Th scope="row">Alpha</Th>
          <Td {...extra}>42</Td>
        </tr>
      </tbody>
    </Table>,
  )
}

describe('Table', () => {
  it('is a real table with column and row headers', () => {
    setup()
    expect(screen.getByRole('table', { name: 'Figures' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument()
    expect(screen.getByRole('rowheader', { name: 'Alpha' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: '42' })).toBeInTheDocument()
  })

  it('passes native attributes through to the cells', () => {
    render(
      <table>
        <tbody>
          <tr>
            <Td colSpan={2} data-testid="wide">x</Td>
            <Th scope="row" className="custom">y</Th>
          </tr>
        </tbody>
      </table>,
    )
    expect(screen.getByTestId('wide')).toHaveAttribute('colspan', '2')
    expect(screen.getByRole('rowheader')).toHaveClass('custom')
  })

  it('marks numeric cells so they can be aligned, and only those', () => {
    setup({ numeric: true })
    expect(screen.getByRole('cell', { name: '42' })).toHaveAttribute('data-numeric', 'true')
    expect(screen.getByRole('columnheader', { name: 'Total' })).toHaveAttribute('data-numeric', 'true')
    expect(screen.getByRole('rowheader', { name: 'Alpha' })).not.toHaveAttribute('data-numeric')
  })
})
