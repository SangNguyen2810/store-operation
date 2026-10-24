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
}

const COLORS = ['var(--series-1)', 'var(--series-2)', 'var(--series-3)', 'var(--series-4)']

/** The only file that knows about the chart library. Swap it here and nothing else changes. */
export function TimeSeriesChart({ rows, series, ariaLabel, formatX, formatY }: Props) {
  return (
    <>
      <div className="chart" role="img" aria-label={ariaLabel}>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={rows} margin={{ top: 8, right: 16, bottom: 0, left: 8 }}>
            <CartesianGrid stroke="var(--grid)" vertical={false} />
            <XAxis dataKey="x" tickFormatter={formatX} stroke="var(--muted)" tickMargin={8} minTickGap={24} />
            <YAxis tickFormatter={formatY} stroke="var(--muted)" width={64} />
            <Tooltip
              labelFormatter={(x) => formatX(String(x))}
              formatter={(v) => formatY(Number(v))}
              contentStyle={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
            />
            <Legend />
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
