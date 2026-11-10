import { cn } from '@/lib/utils'
import type { Stage } from '@/api/battles'
import { stages } from '@/lib/stages'

type StemMeterProps = {
  current?: Stage
  done?: boolean
  className?: string
}

export function StemMeter({ current, done, className }: StemMeterProps) {
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
            <span className="text-sm">{stage.label}</span>
            <span className="font-mono">{stage.points}</span>
          </li>
        )
      })}
    </ol>
  )
}
