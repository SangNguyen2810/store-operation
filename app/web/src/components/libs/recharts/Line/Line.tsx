import { Line as RechartsLine } from 'recharts'
import type { ComponentProps } from 'react'

export type LineProps = ComponentProps<typeof RechartsLine>

/** A plain 2px line with no dots and no animation, so charts render the same every time. */
export function Line(props: LineProps) {
  return <RechartsLine strokeWidth={2} dot={false} isAnimationActive={false} {...props} />
}
