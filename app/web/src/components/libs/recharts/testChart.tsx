import { render } from '@testing-library/react'
import type { ReactNode } from 'react'
import { LineChart } from './LineChart'

export const data = [
  { x: 'Jun 28', a: 10, b: 30 },
  { x: 'Jun 29', a: 20, b: 15 },
  { x: 'Jun 30', a: 15, b: 25 },
]

/** jsdom cannot lay out a ResponsiveContainer, so tests draw into a chart with a fixed size. */
export function renderChart(children: ReactNode) {
  return render(
    <LineChart width={400} height={240} data={data}>
      {children}
    </LineChart>,
  )
}
