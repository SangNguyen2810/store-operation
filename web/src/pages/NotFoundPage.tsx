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
    <section className="not-found" aria-labelledby="not-found-title">
      <p className="not-found-code" aria-hidden="true">404</p>
      <h1 id="not-found-title">Page not found</h1>
      <p className="muted">The page you are looking for does not exist or has moved.</p>
      <Link to="/" className="button-link">Back to the dashboard</Link>
    </section>
  )
}
