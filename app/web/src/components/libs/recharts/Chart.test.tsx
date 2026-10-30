import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CartesianGrid, Legend, Line, LineChart, Tooltip, XAxis, YAxis } from '.'
import { data } from './testChart'

describe('recharts wrappers together', () => {
  it('compose into a complete chart: lines, axes, grid and legend', () => {
    const { container } = render(
      <LineChart width={480} height={260} data={data}>
        <CartesianGrid />
        <XAxis dataKey="x" />
        <YAxis />
        <Tooltip />
        <Legend />
        <Line dataKey="a" name="Store A" stroke="var(--series-1)" />
        <Line dataKey="b" name="Store B" stroke="var(--series-2)" />
      </LineChart>,
    )
    expect(container.querySelectorAll('.recharts-line-curve')).toHaveLength(2)
    expect(container.querySelector('.recharts-cartesian-grid')).not.toBeNull()
    expect(container.querySelector('.recharts-xAxis')).not.toBeNull()
    expect(container.querySelector('.recharts-yAxis')).not.toBeNull()
    expect([...container.querySelectorAll('.recharts-legend-item-text')].map((t) => t.textContent)).toEqual(['Store A', 'Store B'])
  })

  it('exposes exactly the eight pieces the app uses', async () => {
    const api = await import('.')
    expect(Object.keys(api).sort()).toEqual(
      ['CartesianGrid', 'Legend', 'Line', 'LineChart', 'ResponsiveContainer', 'Tooltip', 'XAxis', 'YAxis'],
    )
  })
})
