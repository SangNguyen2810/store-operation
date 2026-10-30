import { Legend as RechartsLegend } from 'recharts'
import type { ComponentProps } from 'react'

export type LegendProps = ComponentProps<typeof RechartsLegend>

export function Legend(props: LegendProps) {
  return <RechartsLegend {...props} />
}
