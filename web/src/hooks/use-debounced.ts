import { useEffect, useState } from 'react'

export function useDebounced<T>(value: T, ms: number) {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), ms)
    return () => window.clearTimeout(timer)
  }, [value, ms])

  return debounced
}
