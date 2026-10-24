import { describe, expect, it } from 'vitest'
import { Line } from '../Line'
import { renderChart } from '../testChart'
import { Tooltip } from './Tooltip'

const box = (c: HTMLElement) => c.querySelector<HTMLElement>('.recharts-default-tooltip')

function chartWith(tooltip: React.ReactElement) {
  return renderChart(
    <>
      {tooltip}
      <Line dataKey="a" name="Store A" />
    </>,
  )
}

describe('Tooltip', () => {
  it('shows the label and value for the active point', () => {
    const { container } = chartWith(<Tooltip active defaultIndex={1} />)
    expect(box(container)).toHaveTextContent('Store A')
    expect(box(container)).toHaveTextContent('20')
  })

  it('uses the theme surface and border colours', () => {
    const { container } = chartWith(<Tooltip active defaultIndex={1} />)
    const style = box(container)!.getAttribute('style') ?? ''
    expect(style).toContain('var(--surface)')
    expect(style).toContain('var(--border)')
  })

  it('merges the caller style over the defaults instead of replacing them', () => {
    const { container } = chartWith(<Tooltip active defaultIndex={1} contentStyle={{ border: '2px solid red' }} />)
    const style = box(container)!.getAttribute('style') ?? ''
    expect(style).toContain('var(--surface)')
    expect(style).toContain('2px solid red')
    expect(style).not.toContain('var(--border)')
  })

  it('formats the value with the caller formatter', () => {
    const { container } = chartWith(<Tooltip active defaultIndex={1} formatter={(v) => `$${v}`} />)
    expect(box(container)).toHaveTextContent('$20')
  })
})
