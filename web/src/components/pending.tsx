import type { ReactNode } from 'react'

export function Pending({ children }: { children: ReactNode }) {
  return (
    <p role="status" className="pending">
      <span className="status-dot" aria-hidden />
      <span>{children}</span>
    </p>
  )
}
