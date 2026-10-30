import { describe, expect, it } from 'vitest'
import { renderChart } from '../testChart'
import { Line } from './Line'

const curve = (c: HTMLElement) => c.querySelector('.recharts-line-curve')

describe('Line', () => {
  it('draws the series as a path', () => {
    const { container } = renderChart(<Line dataKey="a" stroke="red" />)
    expect(curve(container)).toHaveAttribute('stroke', 'red')
    expect(curve(container)?.getAttribute('d')).toMatch(/^M/)
  })

  it('is 2px wide by default, and the caller can override it', () => {
    expect(curve(renderChart(<Line dataKey="a" />).container)).toHaveAttribute('stroke-width', '2')
    expect(curve(renderChart(<Line dataKey="a" strokeWidth={4} />).container)).toHaveAttribute('stroke-width', '4')
  })

  it('has no dots by default, and the caller can turn them on', () => {
    expect(renderChart(<Line dataKey="a" />).container.querySelectorAll('.recharts-line-dot')).toHaveLength(0)
    expect(renderChart(<Line dataKey="a" dot />).container.querySelectorAll('.recharts-line-dot')).toHaveLength(3)
  })

  it('draws immediately: animation is off by default', () => {
    const { container } = renderChart(<Line dataKey="a" />)
    expect(curve(container)).not.toHaveAttribute('stroke-dasharray')
  })
})
