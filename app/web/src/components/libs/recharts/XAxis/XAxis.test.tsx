import { describe, expect, it } from 'vitest'
import { Line } from '../Line'
import { renderChart } from '../testChart'
import { XAxis } from './XAxis'

const axisLine = (c: HTMLElement) => c.querySelector('.recharts-xAxis .recharts-cartesian-axis-line')
// Recharts draws tick labels in their own layer, next to (not inside) the axis group.
const labels = (c: HTMLElement) => [...c.querySelectorAll('.recharts-xAxis-tick-labels .recharts-cartesian-axis-tick-value')].map((t) => t.textContent)

describe('XAxis', () => {
  it('shows a tick label for the data key', () => {
    const { container } = renderChart(
      <>
        <XAxis dataKey="x" />
        <Line dataKey="a" />
      </>,
    )
    expect(labels(container)).toContain('Jun 28')
  })

  it('uses the theme muted colour for the axis line', () => {
    const { container } = renderChart(<XAxis dataKey="x" />)
    expect(axisLine(container)).toHaveAttribute('stroke', 'var(--muted)')
  })

  it('lets the caller override the colour and format the labels', () => {
    const { container } = renderChart(<XAxis dataKey="x" stroke="green" tickFormatter={(v) => `<${v}>`} />)
    expect(axisLine(container)).toHaveAttribute('stroke', 'green')
    expect(labels(container)).toContain('<Jun 28>')
  })

  it('leaves 8px between the ticks and the axis line by default', () => {
    const { container } = renderChart(<XAxis dataKey="x" />)
    const tick = container.querySelector('.recharts-xAxis-tick-labels .recharts-cartesian-axis-tick-value')!
    const withMargin = Number(tick.getAttribute('y'))
    const other = renderChart(<XAxis dataKey="x" tickMargin={0} />).container
    const without = Number(other.querySelector('.recharts-xAxis-tick-labels .recharts-cartesian-axis-tick-value')!.getAttribute('y'))
    expect(withMargin - without).toBe(8)
  })
})
