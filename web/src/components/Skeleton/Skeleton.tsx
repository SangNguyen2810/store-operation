export function Skeleton({ height = 16, width = '100%' }: { height?: number; width?: number | string }) {
  return (
    <div
      className="animate-skeleton rounded-md bg-skeleton motion-reduce:animate-none"
      style={{ height, width }}
      aria-hidden="true"
    />
  )
}
