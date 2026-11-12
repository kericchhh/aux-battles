import type { Stage } from '@/api/battles'
import { cn } from '@/lib/utils'

import { stages } from '@/components/game/stages'

type StemMeterProps = {
  current?: Stage
  done?: boolean
  playing?: boolean
  className?: string
}

export function StemMeter({ current, done, playing, className }: StemMeterProps) {
  const active = stages.findIndex((s) => s.id === current)

  return (
    <ol className={cn('stem-meter', className)}>
      {stages.map((stage, i) => {
        const state = done || i < active ? 'past' : i === active ? 'now' : 'next'
        return (
          <li
            key={stage.id}
            data-state={state}
            aria-current={state === 'now' ? 'step' : undefined}
            className="stem-cell"
          >
            <span className="flex items-center justify-between gap-2 text-sm">
              {stage.label}
              {state === 'now' && (
                <span className="levels" data-playing={playing || undefined} aria-hidden>
                  <i />
                  <i />
                  <i />
                  <i />
                </span>
              )}
            </span>
            <span className="stem-points">{stage.points}</span>
          </li>
        )
      })}
    </ol>
  )
}
