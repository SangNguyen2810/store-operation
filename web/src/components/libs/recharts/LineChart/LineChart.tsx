import { LineChart as RechartsLineChart } from 'recharts'
import type { ComponentProps } from 'react'

export type LineChartProps = ComponentProps<typeof RechartsLineChart>

export function LineChart(props: LineChartProps) {
  return <RechartsLineChart {...props} />
}
