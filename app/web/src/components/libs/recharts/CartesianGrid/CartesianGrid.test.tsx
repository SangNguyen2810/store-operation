import { describe, expect, it } from 'vitest'
import { Line } from '../Line'
import { renderChart } from '../testChart'
import { CartesianGrid } from './CartesianGrid'

const lines = (c: HTMLElement, kind: 'horizontal' | 'vertical') =>
  c.querySelectorAll(`.recharts-cartesian-grid-${kind} line`)

describe('CartesianGrid', () => {
  it('draws horizontal guides only, by default', () => {
    const { container } = renderChart(
      <>
        <CartesianGrid />
        <Line dataKey="a" />
      </>,
    )
    expect(lines(container, 'horizontal').length).toBeGreaterThan(0)
    expect(lines(container, 'vertical')).toHaveLength(0)
  })

  it('uses the theme grid colour', () => {
    const { container } = renderChart(<CartesianGrid />)
    expect(lines(container, 'horizontal')[0]).toHaveAttribute('stroke', 'var(--grid)')
  })

  it('lets the caller override the colour and bring back vertical guides', () => {
    const { container } = renderChart(<CartesianGrid stroke="blue" vertical />)
    expect(lines(container, 'horizontal')[0]).toHaveAttribute('stroke', 'blue')
    expect(lines(container, 'vertical').length).toBeGreaterThan(0)
  })
})
