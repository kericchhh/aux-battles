import { useQuery } from '@tanstack/react-query'

import { statsQuery, type Stats } from '@/api/battles'
import { DataState } from '@/components/data-state'
import { Card } from '@/components/ui/card'

const labels: [keyof Stats, string][] = [
  ['played', 'Played'],
  ['wins', 'Wins'],
  ['losses', 'Losses'],
  ['draws', 'Draws'],
  ['bestScore', 'Best score'],
  ['averageScore', 'Average score'],
]

export function StatsCard() {
  const stats = useQuery(statsQuery)

  return (
    <Card title="Your stats" aria-busy={stats.isPending}>
      <DataState
        query={{ isPending: false, isError: stats.isError, data: stats.data }}
        loading=""
        error="Could not load your stats."
      >
        <dl className="stats">
          {labels.map(([key, label]) => (
            <div key={key} className="stat">
              <dt>{label}</dt>
              <dd>{stats.data?.[key] ?? '–'}</dd>
            </div>
          ))}
        </dl>
      </DataState>
    </Card>
  )
}
