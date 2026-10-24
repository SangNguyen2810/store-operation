import type { FilmStat } from '../../api/types'
import { Skeleton } from '../../components/Skeleton'
import { formatCurrency, formatInteger } from '../../lib/format'

export function TopFilms({ films }: { films: FilmStat[] }) {
  const max = Math.max(...films.map((f) => f.revenue), 1)
  return (
    <table className="top-films">
      <thead>
        <tr>
          <th scope="col">#</th>
          <th scope="col">Film</th>
          <th scope="col" className="num">Rentals</th>
          <th scope="col" className="num">Revenue</th>
        </tr>
      </thead>
      <tbody>
        {films.map((f, i) => (
          <tr key={f.filmId}>
            <td>{i + 1}</td>
            <th scope="row">
              {f.title}
              <span className="bar" style={{ width: `${(f.revenue / max) * 100}%` }} aria-hidden="true" />
            </th>
            <td className="num">{formatInteger(f.rentals)}</td>
            <td className="num">{formatCurrency(f.revenue)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export function TopFilmsSkeleton() {
  return (
    <div className="stack">
      {Array.from({ length: 5 }, (_, i) => (
        <Skeleton key={i} height={20} />
      ))}
    </div>
  )
}
