import { useEffect } from 'react'
import { Link } from 'react-router-dom'

export function NotFoundPage() {
  useEffect(() => {
    const previous = document.title
    document.title = 'Page not found · Store Operations'
    return () => {
      document.title = previous
    }
  }, [])

  return (
    <section
      aria-labelledby="not-found-title"
      className="grid justify-items-center gap-2 rounded-[10px] border border-line bg-surface px-4 py-12 text-center"
    >
      <p className="m-0 text-[3.5rem] leading-none font-bold text-muted" aria-hidden="true">404</p>
      <h1 id="not-found-title" className="m-0 text-[1.35rem] font-bold sm:text-[1.6rem]">Page not found</h1>
      <p className="mt-1 text-muted">The page you are looking for does not exist or has moved.</p>
      <Link
        to="/"
        className="mt-2 inline-flex min-h-(--control-h) items-center rounded-ui bg-accent px-4 font-semibold text-page no-underline"
      >
        Back to the dashboard
      </Link>
    </section>
  )
}
