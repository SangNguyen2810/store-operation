import { CartesianGrid as RechartsCartesianGrid } from 'recharts'
import type { ComponentProps } from 'react'

export type CartesianGridProps = ComponentProps<typeof RechartsCartesianGrid>

/** Horizontal guide lines only, in the theme's grid colour. */
export function CartesianGrid(props: CartesianGridProps) {
  return <RechartsCartesianGrid stroke="var(--grid)" vertical={false} {...props} />
}
