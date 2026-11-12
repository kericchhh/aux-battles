import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useState, type CSSProperties } from 'react'

import { battleKey, type Battle } from '@/api/battles'
import { useCountdown } from '@/hooks/use-countdown'
import { countLabel } from '@/lib/format'

type CountdownProps = { battle: Battle; endsAt: string; seconds: number; receivedAt: number }

export function Countdown({ battle, endsAt, seconds: total, receivedAt }: CountdownProps) {
  const queryClient = useQueryClient()
  const remaining = useCountdown(endsAt, battle.serverTime, receivedAt) ?? 0
  const seconds = Math.ceil(remaining / 1000)
  const expired = remaining === 0
  const warningText = seconds > 0 && seconds <= 10 ? `${countLabel(seconds, 'second')} left.` : ''
  const [warning, setWarning] = useState({ endsAt, text: warningText })
  if (warning.endsAt !== endsAt || (!warning.text && warningText)) {
    setWarning({ endsAt, text: warningText })
  }

  useEffect(() => {
    if (!expired) return
    const timer = window.setInterval(() => {
      void queryClient.invalidateQueries(
        { queryKey: battleKey(battle.id) },
        { cancelRefetch: false },
      )
    }, 2500)
    return () => window.clearInterval(timer)
  }, [expired, battle.id, queryClient])

  return (
    <div className="grid gap-2">
      <div className="flex items-baseline justify-between">
        <span className="label">Time left</span>
        <span
          role="timer"
          aria-live="off"
          className="timer"
          data-numeric={!expired || undefined}
          data-urgent={seconds <= 5 || undefined}
        >
          {expired ? 'Time’s up' : `0:${String(seconds).padStart(2, '0')}`}
        </span>
      </div>
      <div
        className="timer-track"
        style={{ '--progress': remaining / (total * 1000) } as CSSProperties}
        aria-hidden
      >
        <span className="timer-fill" data-urgent={seconds <= 5 || undefined} />
      </div>
      <p role="status" className="sr-only">
        {expired ? 'Time is up on this stem.' : warning?.endsAt === endsAt ? warning.text : ''}
      </p>
    </div>
  )
}
