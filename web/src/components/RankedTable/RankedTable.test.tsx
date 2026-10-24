import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { RankedTable, RankedTableSkeleton } from './RankedTable'

const columns = [{ label: 'Rentals', collapsible: true }, { label: 'Revenue' }]
const rows = [
  { id: 1, label: 'FOOL MOCKINGBIRD', meta: '7 rentals', bar: 1, values: ['7', '$52.93'] },
  { id: 2, label: 'DORADO NOTTING', meta: '8 rentals', bar: 0.5, values: ['8', '$51.92'] },
]

describe('RankedTable', () => {
  it('is a table with a header per column and a numbered row per item', () => {
    render(<RankedTable labelHeader="Film" columns={columns} rows={rows} />)
    const table = screen.getByRole('table')
    for (const name of ['#', 'Film', 'Rentals', 'Revenue']) {
      expect(within(table).getByRole('columnheader', { name })).toBeInTheDocument()
    }
    const first = within(table).getAllByRole('row')[1]
    expect(within(first).getByText('1')).toBeInTheDocument()
    expect(within(first).getByRole('rowheader')).toHaveTextContent('FOOL MOCKINGBIRD')
    expect(within(first).getByText('$52.93')).toBeInTheDocument()
  })

  it('marks collapsible columns so small screens can hide them', () => {
    render(<RankedTable labelHeader="Film" columns={columns} rows={rows} />)
    expect(screen.getByRole('columnheader', { name: 'Rentals' })).toHaveClass('col-collapsible')
    expect(screen.getByRole('columnheader', { name: 'Revenue' })).not.toHaveClass('col-collapsible')
  })

  it('sizes the bar relative to the largest row and clamps out-of-range values', () => {
    const { container } = render(
      <RankedTable labelHeader="Film" columns={columns} rows={[{ ...rows[0], bar: 1.4 }, { ...rows[1], bar: 0.5 }, { ...rows[1], id: 3, bar: -1 }]} />,
    )
    const widths = [...container.querySelectorAll<HTMLElement>('.ranked-bar')].map((b) => b.style.width)
    expect(widths).toEqual(['100%', '50%', '0%'])
  })

  it('omits the meta line when there is none', () => {
    const { container } = render(<RankedTable labelHeader="Film" columns={columns} rows={[{ ...rows[0], meta: undefined }]} />)
    expect(container.querySelector('.ranked-meta')).toBeNull()
  })

  it('renders a skeleton with the requested number of rows', () => {
    const { container } = render(<RankedTableSkeleton rows={3} />)
    expect(container.querySelectorAll('.skeleton')).toHaveLength(3)
  })
})
