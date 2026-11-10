export function PageLoader({ label = 'Loading' }: { label?: string }) {
  return (
    <div role="status" className="grid min-h-[50dvh] place-items-center">
      <span className="flex items-center gap-3 text-dim">
        <span className="size-3 animate-pulse bg-accent" aria-hidden />
        {label}…
      </span>
    </div>
  )
}
