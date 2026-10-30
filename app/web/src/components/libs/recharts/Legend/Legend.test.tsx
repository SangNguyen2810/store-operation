import { describe, expect, it } from 'vitest'
import { Line } from '../Line'
import { renderChart } from '../testChart'
import { Legend } from './Legend'

const items = (c: HTMLElement) => [...c.querySelectorAll('.recharts-legend-item-text')].map((t) => t.textContent)

describe('Legend', () => {
  it('lists each series by its name', () => {
    const { container } = renderChart(
      <>
        <Legend />
        <Line dataKey="a" name="Store A" />
        <Line dataKey="b" name="Store B" />
      </>,
    )
    expect(items(container)).toEqual(['Store A', 'Store B'])
  })

  it('passes styling through to the legend wrapper', () => {
    const { container } = renderChart(
      <>
        <Legend wrapperStyle={{ fontSize: 13 }} />
        <Line dataKey="a" name="Store A" />
      </>,
    )
    expect(container.querySelector<HTMLElement>('.recharts-legend-wrapper')?.style.fontSize).toBe('13px')
  })
})
