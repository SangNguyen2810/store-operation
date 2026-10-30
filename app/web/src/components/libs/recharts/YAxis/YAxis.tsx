import { YAxis as RechartsYAxis } from 'recharts'
import type { ComponentProps } from 'react'

export type YAxisProps = ComponentProps<typeof RechartsYAxis>

export function YAxis(props: YAxisProps) {
  return <RechartsYAxis stroke="var(--muted)" {...props} />
}
