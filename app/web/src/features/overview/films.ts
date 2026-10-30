import type { FilmStat } from '../../api/types'
import type { RankedColumn, RankedRow } from '../../components/RankedTable'
import { formatCurrency, formatInteger } from '../../lib/format'

export const FILM_COLUMNS: RankedColumn[] = [{ label: 'Rentals', collapsible: true }, { label: 'Revenue' }]

/** Rows are ranked as given; each bar is relative to the highest-earning film. */
export function toFilmRows(films: FilmStat[]): RankedRow[] {
  const max = Math.max(...films.map((f) => f.revenue), 1)
  return films.map((f) => ({
    id: f.filmId,
    label: f.title,
    meta: `${formatInteger(f.rentals)} rentals`,
    bar: f.revenue / max,
    values: [formatInteger(f.rentals), formatCurrency(f.revenue)],
  }))
}
