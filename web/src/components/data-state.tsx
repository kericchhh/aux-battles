import type { ReactNode } from 'react'

type DataStateProps = {
  query: { isPending: boolean; isError: boolean; data?: unknown }
  loading: string
  error: string
  empty?: ReactNode
  children: ReactNode
}

export function DataState({ query, loading, error, empty, children }: DataStateProps) {
  if (query.isPending) return <p className="text-dim">{loading}</p>
  if (query.isError && query.data === undefined) {
    return (
      <p role="alert" className="text-alarm">
        {error}
      </p>
    )
  }
  return empty || children
}
