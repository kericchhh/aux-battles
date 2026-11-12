import { opponentName, type Battle } from '@/api/battles'

export function Scoreboard({ battle }: { battle: Battle }) {
  const opponent = opponentName(battle, 'Opponent')
  const lead = Math.sign(battle.myScore - battle.opponentScore)

  return (
    <section aria-label="Score" className="scoreboard">
      <div className="score" data-leading={lead > 0 || undefined}>
        <span className="score-name">{battle.me.username}</span>
        <span className="score-value">{battle.myScore}</span>
      </div>
      <p className="score-round">
        {battle.status === 'ONGOING' ? (
          <>
            Round <span className="font-mono tabular">{battle.currentRound}</span> of{' '}
            <span className="font-mono tabular">{battle.rounds}</span>
          </>
        ) : (
          'Final score'
        )}
      </p>
      <div className="score" data-side="opponent" data-leading={lead < 0 || undefined}>
        <span className="score-name">{opponent}</span>
        <span className="score-value">{battle.opponentScore}</span>
      </div>
    </section>
  )
}
