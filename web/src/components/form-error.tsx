import { TriangleAlert } from 'lucide-react'

export function FormError({ message }: { message?: string | null }) {
  return (
    <div role="alert" className="live">
      {message && (
        <p className="flex animate-rise items-start gap-2 text-alarm">
          <TriangleAlert aria-hidden className="mt-0.5 size-4 shrink-0" />
          {message}
        </p>
      )}
    </div>
  )
}
