import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Line } from '../Line'
import { data } from '../testChart'
import { LineChart } from './LineChart'

describe('LineChart', () => {
  it('draws an svg of the requested size', () => {
    const { container } = render(
      <LineChart width={320} height={200} data={data}>
        <Line dataKey="a" />
      </LineChart>,
    )
    const svg = container.querySelector('svg.recharts-surface')
    expect(svg).toHaveAttribute('width', '320')
    expect(svg).toHaveAttribute('height', '200')
  })

  it('draws one line per child Line', () => {
    const { container } = render(
      <LineChart width={320} height={200} data={data}>
        <Line dataKey="a" />
        <Line dataKey="b" />
      </LineChart>,
    )
    expect(container.querySelectorAll('.recharts-line-curve')).toHaveLength(2)
  })

  it('draws no line for an empty data set', () => {
    const { container } = render(
      <LineChart width={320} height={200} data={[]}>
        <Line dataKey="a" />
      </LineChart>,
    )
    expect(container.querySelector('.recharts-line-curve')?.getAttribute('d') ?? '').toBe('')
  })
})
