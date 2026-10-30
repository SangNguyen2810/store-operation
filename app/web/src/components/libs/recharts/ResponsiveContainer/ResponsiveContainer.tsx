import { ResponsiveContainer as RechartsResponsiveContainer } from 'recharts'
import type { ComponentProps } from 'react'

export type ResponsiveContainerProps = ComponentProps<typeof RechartsResponsiveContainer>

/** Fills the width of its parent (Recharts' own default); give it a height. */
export function ResponsiveContainer(props: ResponsiveContainerProps) {
  return <RechartsResponsiveContainer {...props} />
}
