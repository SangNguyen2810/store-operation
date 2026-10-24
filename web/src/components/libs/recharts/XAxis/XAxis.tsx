import { XAxis as RechartsXAxis } from 'recharts'
import type { ComponentProps } from 'react'

export type XAxisProps = ComponentProps<typeof RechartsXAxis>

/** Themed axis with room between ticks and the line, and a minimum gap so labels never collide. */
export function XAxis(props: XAxisProps) {
  return <RechartsXAxis stroke="var(--muted)" tickMargin={8} minTickGap={32} {...props} />
}
