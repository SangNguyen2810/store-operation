import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { TimeSeriesChart } from './TimeSeriesChart'

const rows = [
  { x: '2022-07-26', a: 10, b: 20 },
  { x: '2022-07-27', a: 15, b: 5 },
]
const series = [
  { key: 'a', label: 'Store A' },
  { key: 'b', label: 'Store B' },
]

function setup() {
  return render(
    <TimeSeriesChart
      rows={rows}
      series={series}
      ariaLabel="Revenue per day"
      formatX={(x) => `day ${x.slice(-2)}`}
      formatY={(v) => `$${v}`}
    />,
  )
}

// jsdom cannot lay out the SVG, so these tests cover the accessible text and the data table.
describe('TimeSeriesChart', () => {
  it('gives the chart a text alternative', () => {
    setup()
    expect(screen.getByRole('img', { name: 'Revenue per day' })).toBeInTheDocument()
  })

  it('offers the data as a table with one column per series', () => {
    setup()
    const table = screen.getByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'Store A' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Store B' })).toBeInTheDocument()
    expect(within(table).getByRole('rowheader', { name: 'day 26' })).toBeInTheDocument()
    expect(within(table).getByText('$15')).toBeInTheDocument()
  })
})
