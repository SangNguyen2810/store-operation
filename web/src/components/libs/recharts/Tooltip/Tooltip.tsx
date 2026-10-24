import { Tooltip as RechartsTooltip } from 'recharts'
import type { ComponentProps } from 'react'

export type TooltipProps = ComponentProps<typeof RechartsTooltip>

const CONTENT_STYLE = { background: 'var(--surface)', border: '1px solid var(--border)' }

/** Uses the surface and border colours, so it follows light and dark mode. Style overrides merge in. */
export function Tooltip({ contentStyle, ...props }: TooltipProps) {
  return <RechartsTooltip contentStyle={{ ...CONTENT_STYLE, ...contentStyle }} {...props} />
}
