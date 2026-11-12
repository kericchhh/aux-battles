import { Link } from 'react-router'

import { opponentName, type Battle } from '@/api/battles'
import { ScoreLine } from '@/components/score-line'
import { RoundSongs } from '@/components/game/round-songs'
import { countLabel } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

function headline(battle: Battle) {
  if (battle.status === 'CANCELLED') return 'Battle cancelled'
  if (battle.outcome === 'WIN') return 'You won'
  if (battle.outcome === 'LOSS') return 'You lost'
  return 'It’s a draw'
}

function reason(battle: Battle) {
  const opponent = opponentName(battle, 'Your opponent')
  if (battle.status === 'CANCELLED') {
    return battle.endReason === 'FORFEIT'
      ? 'The lobby was cancelled before anyone joined.'
      : 'Nobody made a move in time, so the battle does not count.'
  }
  if (battle.endReason === 'FORFEIT') {
    return battle.outcome === 'WIN' ? `${opponent} forfeited.` : 'You forfeited.'
  }
  if (battle.endReason === 'TIMEOUT') {
    return battle.outcome === 'WIN'
      ? `${opponent} stopped playing, so you win.`
      : 'You were inactive for too long.'
  }
  return `All ${countLabel(battle.rounds, 'round')} played.`
}

export function Results({ battle }: { battle: Battle }) {
  return (
    <div className="grid gap-5">
      <Card
        title={<span className="result-title">{headline(battle)}</span>}
        description={reason(battle)}
        action={
          <Button asChild>
            <Link to="/">Play again</Link>
          </Button>
        }
      />
      {battle.completedRounds.length > 0 && (
        <Card title="Rounds">
          <ol className="grid gap-1">
            {battle.completedRounds.map((round) => (
              <li key={round.number} className="result-row">
                <span className="font-mono text-dim tabular">{round.number}</span>
                <RoundSongs round={round} className="sm:grid-cols-2" />
                <ScoreLine mine={round.myPoints} theirs={round.opponentPoints} />
              </li>
            ))}
          </ol>
        </Card>
      )}
    </div>
  )
}
