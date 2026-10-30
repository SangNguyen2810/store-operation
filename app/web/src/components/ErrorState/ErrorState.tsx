interface Props {
  message: string
  onRetry: () => void
}

/** A failed section or page, with a way to try again. */
export function ErrorState({ message, onRetry }: Props) {
  return (
    <div className="m-0 rounded-lg border border-down p-4" role="alert">
      <p className="mb-2">{message}</p>
      <button type="button" onClick={onRetry}>
        Retry
      </button>
    </div>
  )
}
