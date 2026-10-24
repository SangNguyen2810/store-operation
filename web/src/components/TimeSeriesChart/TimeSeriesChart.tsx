import { useBreakpoint } from '../../lib/useBreakpoint'
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

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

/** The only file that knows about the chart library. Swap it here and nothing else changes. */
export function TimeSeriesChart({ rows, series, ariaLabel, formatX, formatY, formatTick = formatY }: Props) {
  const size = SIZES[useBreakpoint()]
  const tick = { fontSize: size.tick }
  return (
    <>
      <div className="chart" role="img" aria-label={ariaLabel}>
        <ResponsiveContainer width="100%" height={size.height}>
          <LineChart data={rows} margin={{ top: 8, right: size.right, bottom: 0, left: size.left }}>
            <CartesianGrid stroke="var(--grid)" vertical={false} />
            <XAxis dataKey="x" tickFormatter={formatX} stroke="var(--muted)" tick={tick} tickMargin={8} minTickGap={32} />
            <YAxis tickFormatter={formatTick} stroke="var(--muted)" tick={tick} width={size.yAxis} />
            <Tooltip
              labelFormatter={(x) => formatX(String(x))}
              formatter={(v) => formatY(Number(v))}
              contentStyle={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
            />
            <Legend wrapperStyle={{ fontSize: size.tick + 1 }} />
            {series.map((s, i) => (
              <Line
                key={s.key}
                dataKey={s.key}
                name={s.label}
                stroke={COLORS[i % COLORS.length]}
                strokeWidth={2}
                dot={false}
                isAnimationActive={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
      <details className="data-table">
        <summary>Show data as a table</summary>
        <table>
          <thead>
            <tr>
              <th scope="col">Date</th>
              {series.map((s) => (
                <th key={s.key} scope="col">
                  {s.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.x}>
                <th scope="row">{formatX(r.x)}</th>
                {series.map((s) => (
                  <td key={s.key}>{formatY(Number(r[s.key]))}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </>
  )
}
