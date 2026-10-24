interface Props {
  message: string
  onRetry: () => void
}

/** A failed section or page, with a way to try again. */
export function ErrorState({ message, onRetry }: Props) {
  return (
    <div className="state state-error" role="alert">
      <p>{message}</p>
      <button type="button" onClick={onRetry}>
        Retry
      </button>
    </div>
  )
}
