import { describe, expect, it } from 'vitest'
import { Line } from '../Line'
import { renderChart } from '../testChart'
import { YAxis } from './YAxis'

const axisLine = (c: HTMLElement) => c.querySelector('.recharts-yAxis .recharts-cartesian-axis-line')
// Recharts draws tick labels in their own layer, next to (not inside) the axis group.
const labels = (c: HTMLElement) => [...c.querySelectorAll('.recharts-yAxis-tick-labels .recharts-cartesian-axis-tick-value')].map((t) => t.textContent)

describe('YAxis', () => {
  it('shows numeric tick labels for the series', () => {
    const { container } = renderChart(
      <>
        <YAxis />
        <Line dataKey="a" />
      </>,
    )
    expect(labels(container).length).toBeGreaterThan(1)
  })

  it('uses the theme muted colour for the axis line', () => {
    expect(axisLine(renderChart(<YAxis />).container)).toHaveAttribute('stroke', 'var(--muted)')
  })

  it('lets the caller override the colour and format the labels', () => {
    const { container } = renderChart(
      <>
        <YAxis stroke="green" tickFormatter={(v) => `$${v}`} />
        <Line dataKey="a" />
      </>,
    )
    expect(axisLine(container)).toHaveAttribute('stroke', 'green')
    expect(labels(container).every((l) => l?.startsWith('$'))).toBe(true)
  })
})
