import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router'

import { battlesQuery, isActive, type BattleSummary } from '@/api/battles'
import { countLabel } from '@/lib/format'
import { DataState } from '@/components/data-state'
import { ScoreLine } from '@/components/score-line'
import { Card } from '@/components/ui/card'
import { Rules } from '@/components/game/how-to-play'
import { StatusBadge } from '@/components/game/status-badge'

export function BattlesCard() {
  const battles = useQuery(battlesQuery)
  const empty = battles.data?.length === 0

  return (
    <Card title={empty ? 'How it works' : 'Your battles'} aria-busy={battles.isPending}>
      <DataState
        query={battles}
        loading="Loading battles…"
        error="Could not load your battles."
        empty={empty && <Rules />}
      >
        <ul className="grid gap-1">
          {battles.data?.map((battle) => (
            <li key={battle.id}>
              <BattleRow battle={battle} />
            </li>
          ))}
        </ul>
      </DataState>
    </Card>
  )
}

function BattleRow({ battle }: { battle: BattleSummary }) {
  const scored = battle.status === 'ONGOING' || battle.status === 'FINISHED'

  return (
    <Link to={`/battles/${battle.id}`} className="battle-row">
      <span className="grid min-w-0 gap-0.5">
        <span className="truncate font-medium">
          {battle.opponent
            ? `vs ${battle.opponent.username}`
            : isActive(battle.status)
              ? 'Waiting for an opponent'
              : 'No opponent joined'}
        </span>
        <span className="text-sm text-dim">
          {battle.status === 'ONGOING'
            ? `Round ${battle.currentRound} of ${battle.rounds}`
            : countLabel(battle.rounds, 'round')}
        </span>
      </span>
      {scored && <ScoreLine mine={battle.myScore} theirs={battle.opponentScore} />}
      <StatusBadge status={battle.status} outcome={battle.outcome} />
    </Link>
  )
}
