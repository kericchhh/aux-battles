import { opponentName, type Battle } from '@/api/battles'
import { RoundSongs } from '@/components/game/round-songs'
import { Card } from '@/components/ui/card'

export function RoundReveal({ battle, className }: { battle: Battle; className?: string }) {
  const last = battle.completedRounds.at(-1)
  const opponent = opponentName(battle, 'Your opponent')

  if (!last) {
    return (
      <Card as="aside" title="Opponent" className={className}>
        <p className="text-dim">
          {battle.round?.opponentFinished
            ? `${opponent} has finished this round.`
            : `${opponent} is still guessing.`}{' '}
          Songs are revealed once you both finish a round.
        </p>
      </Card>
    )
  }
  return (
    <Card as="aside" title={`Round ${last.number} reveal`} className={className}>
      <RoundSongs round={last}>
        <div>
          <dt>Points</dt>
          <dd>
            You <span className="font-mono tabular">{last.myPoints}</span>, them{' '}
            <span className="font-mono tabular">{last.opponentPoints}</span>
          </dd>
        </div>
      </RoundSongs>
    </Card>
  )
}
