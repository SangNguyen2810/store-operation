import { useLocation, useNavigate } from 'react-router-dom'

/** Shows the current URL so a test can assert on it. */
export function Location() {
  const { pathname, search } = useLocation()
  return <output data-testid="location">{pathname + search}</output>
}

/** Stands in for the browser's Back button. */
export function BackButton() {
  const navigate = useNavigate()
  return (
    <button type="button" onClick={() => void navigate(-1)}>
      Browser back
    </button>
  )
}
