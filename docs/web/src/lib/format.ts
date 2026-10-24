const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })
const integer = new Intl.NumberFormat('en-US')

export const formatCurrency = (v: number) => currency.format(v)
export const formatInteger = (v: number) => integer.format(v)

/** "+3.2%", "-1.0%", "0.0%". */
export function formatDelta(pct: number): string {
  const sign = pct > 0 ? '+' : ''
  return `${sign}${pct.toFixed(1)}%`
}

/** 2022-07-27 -> "Jul 27". Parsed as UTC so the day never shifts with the viewer's timezone. */
export function formatBucket(iso: string): string {
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(
    new Date(`${iso}T00:00:00Z`),
  )
}
