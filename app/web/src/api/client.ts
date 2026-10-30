export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

/** GETs JSON from the API. Non-2xx responses become ApiError carrying the problem detail. */
export async function getJson<T>(
  path: string,
  params: Record<string, string | number | null | undefined> = {},
  signal?: AbortSignal,
): Promise<T> {
  const qs = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== null && value !== undefined) qs.set(key, String(value))
  }
  const query = qs.toString()
  const res = await fetch(query ? `${path}?${query}` : path, { signal })
  if (!res.ok) {
    let detail = res.statusText
    try {
      const body = (await res.json()) as { detail?: string; title?: string }
      detail = body.detail ?? body.title ?? detail
    } catch {
      // body was not JSON; keep the status text
    }
    throw new ApiError(res.status, detail)
  }
  return (await res.json()) as T
}
