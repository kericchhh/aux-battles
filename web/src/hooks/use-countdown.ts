import { useEffect, useState } from 'react'

export function useCountdown(endsAt: string | null, serverTime: string, receivedAt: number) {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (!endsAt) return
    const timer = window.setInterval(() => setNow(Date.now()), 250)
    return () => window.clearInterval(timer)
  }, [endsAt])

  if (!endsAt) return null
  const offset = Date.parse(serverTime) - receivedAt
  return Math.max(0, Date.parse(endsAt) - (now + offset))
}
