import { Skeleton } from '../Skeleton'
import { Table, Td, Th } from '../Table'

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

/** Collapsible columns are hidden below 640px; `data-collapsible` marks them for tests and tooling. */
const collapse = (c: RankedColumn) => (c.collapsible ? { 'data-collapsible': true, className: 'max-sm:hidden' } : {})

/** A ranked list as a real table: numbered rows, a label with a relative bar, and right-aligned figures. */
export function RankedTable({ labelHeader, columns, rows }: Props) {
  return (
    <Table>
      <thead>
        <tr>
          <Th scope="col">#</Th>
          <Th scope="col">{labelHeader}</Th>
          {columns.map((c) => (
            <Th key={c.label} scope="col" numeric {...collapse(c)}>
              {c.label}
            </Th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => (
          <tr key={r.id}>
            <Td>{i + 1}</Td>
            <Th scope="row" className="relative">
              {r.label}
              {r.meta && <span className="block text-[0.75rem] text-muted sm:hidden">{r.meta}</span>}
              <span
                className="mt-1 block h-[3px] rounded-[2px] bg-(--series-1) opacity-60"
                style={{ width: `${Math.max(0, Math.min(1, r.bar)) * 100}%` }}
                aria-hidden="true"
              />
            </Th>
            {columns.map((c, j) => (
              <Td key={c.label} numeric {...collapse(c)}>
                {r.values[j]}
              </Td>
            ))}
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

export function RankedTableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="grid gap-2.5">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} height={20} />
      ))}
    </div>
  )
}
