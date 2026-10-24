import { Skeleton } from '../Skeleton'

export interface RankedColumn {
  label: string
  /** On small screens this column is hidden and its content moves under the row label (see `meta`). */
  collapsible?: boolean
}

export interface RankedRow {
  id: string | number
  label: string
  /** Shown under the label on small screens, in place of collapsible columns. */
  meta?: string
  /** 0 to 1, drawn as a thin bar relative to the largest row. */
  bar: number
  /** One entry per column. */
  values: string[]
}

interface Props {
  labelHeader: string
  columns: RankedColumn[]
  rows: RankedRow[]
}

/** A ranked list as a real table: numbered rows, a label with a relative bar, and right-aligned figures. */
export function RankedTable({ labelHeader, columns, rows }: Props) {
  const cls = (c: RankedColumn) => (c.collapsible ? 'num col-collapsible' : 'num')
  return (
    <table className="ranked-table">
      <thead>
        <tr>
          <th scope="col">#</th>
          <th scope="col">{labelHeader}</th>
          {columns.map((c) => (
            <th key={c.label} scope="col" className={cls(c)}>
              {c.label}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => (
          <tr key={r.id}>
            <td>{i + 1}</td>
            <th scope="row">
              {r.label}
              {r.meta && <span className="ranked-meta">{r.meta}</span>}
              <span className="ranked-bar" style={{ width: `${Math.max(0, Math.min(1, r.bar)) * 100}%` }} aria-hidden="true" />
            </th>
            {columns.map((c, j) => (
              <td key={c.label} className={cls(c)}>
                {r.values[j]}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export function RankedTableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="stack">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} height={20} />
      ))}
    </div>
  )
}
