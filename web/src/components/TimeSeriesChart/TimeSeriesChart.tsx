import { useBreakpoint } from '../../lib/useBreakpoint'
import { Table, Td, Th } from '../Table'
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from '../libs/recharts'

export interface Series {
  key: string
  label: string
}

export type Row = { x: string } & Record<string, number | string>

interface Props {
  rows: Row[]
  series: Series[]
  ariaLabel: string
  formatX: (x: string) => string
  formatY: (v: number) => string
  /** Shorter format for axis ticks; defaults to formatY. */
  formatTick?: (v: number) => string
}

// Pixel sizes the chart library needs as props; CSS cannot set them.
const SIZES = {
  mobile: { height: 240, yAxis: 48, tick: 11, left: 0, right: 8 },
  tablet: { height: 300, yAxis: 56, tick: 12, left: 8, right: 16 },
  desktop: { height: 380, yAxis: 56, tick: 12, left: 8, right: 16 },
} as const

const COLORS = ['var(--series-1)', 'var(--series-2)', 'var(--series-3)', 'var(--series-4)']

/** Draws a multi-series line chart. The chart library is reached only through ../libs/recharts. */
export function TimeSeriesChart({ rows, series, ariaLabel, formatX, formatY, formatTick = formatY }: Props) {
  const size = SIZES[useBreakpoint()]
  const tick = { fontSize: size.tick }
  return (
    <>
      <div className="w-full" role="img" aria-label={ariaLabel}>
        <ResponsiveContainer width="100%" height={size.height}>
          <LineChart data={rows} margin={{ top: 8, right: size.right, bottom: 0, left: size.left }}>
            <CartesianGrid />
            <XAxis dataKey="x" tickFormatter={formatX} tick={tick} />
            <YAxis tickFormatter={formatTick} tick={tick} width={size.yAxis} />
            <Tooltip labelFormatter={(x) => formatX(String(x))} formatter={(v) => formatY(Number(v))} />
            <Legend wrapperStyle={{ fontSize: size.tick + 1 }} />
            {series.map((s, i) => (
              <Line key={s.key} dataKey={s.key} name={s.label} stroke={COLORS[i % COLORS.length]} />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
      <details className="mt-3">
        <summary className="cursor-pointer py-[calc((var(--control-h)-1.5em)/2)] text-accent">Show data as a table</summary>
        <Table className="mt-2 block max-h-[280px] overflow-auto">
          <thead>
            <tr>
              <Th scope="col">Date</Th>
              {series.map((s) => (
                <Th key={s.key} scope="col">
                  {s.label}
                </Th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.x}>
                <Th scope="row" className="whitespace-nowrap">{formatX(r.x)}</Th>
                {series.map((s) => (
                  <Td key={s.key}>{formatY(Number(r[s.key]))}</Td>
                ))}
              </tr>
            ))}
          </tbody>
        </Table>
      </details>
    </>
  )
}
