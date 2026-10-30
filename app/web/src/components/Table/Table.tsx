import type { ComponentProps } from 'react'

/** Shared look for the app's data tables, so each table only decides its content. */

export function Table({ className = '', ...props }: ComponentProps<'table'>) {
  return <table className={`w-full border-collapse ${className}`} {...props} />
}

type CellProps = { numeric?: boolean }

const CELL = 'border-b border-line px-1.5 py-2 text-left font-normal sm:px-2.5 sm:py-1.5'
const NUMERIC = 'text-right tabular-nums'
const HEADING = 'text-[0.8rem] text-muted'

/** A column header (scope="col") is muted and small; a row header is regular text. */
export function Th({ numeric, className = '', scope, ...props }: ComponentProps<'th'> & CellProps) {
  const style = [CELL, scope === 'col' ? HEADING : '', numeric ? NUMERIC : '', className].filter(Boolean).join(' ')
  return <th scope={scope} data-numeric={numeric || undefined} className={style} {...props} />
}

export function Td({ numeric, className = '', ...props }: ComponentProps<'td'> & CellProps) {
  return <td data-numeric={numeric || undefined} className={[CELL, numeric ? NUMERIC : '', className].filter(Boolean).join(' ')} {...props} />
}
