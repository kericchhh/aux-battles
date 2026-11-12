import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useLayoutEffect, useRef } from 'react'
import { Link, useParams } from 'react-router'

import { meQuery } from '@/api/auth'
import { battlesQuery, isActive, statsQuery, type Battle } from '@/api/battles'
import { isApiError } from '@/api/client'
import { countLabel } from '@/lib/format'
import { PageHeader } from '@/components/page-header'
import { PageLoader } from '@/components/page-loader'
import { Button } from '@/components/ui/button'
import { ForfeitButton } from '@/features/battle/forfeit-button'
import { LineupPicker } from '@/features/battle/lineup-picker'
import { Results } from '@/features/battle/results'
import { RoundPlay } from '@/features/battle/round-play'
import { Scoreboard } from '@/features/battle/scoreboard'
import { StatusBadge } from '@/components/game/status-badge'
import { WaitingRoom } from '@/features/battle/waiting-room'
import { useLiveBattle } from '@/hooks/use-live-battle'

const describe = (battle: Battle) =>
  `${countLabel(battle.rounds, 'round')}, ${
    battle.guessSeconds ? `${battle.guessSeconds} seconds per stem` : 'no timer'
  }`

export function BattlePage() {
  const { battleId = '' } = useParams()
  return <BattleScreen key={battleId} battleId={battleId} />
}

function BattleScreen({ battleId }: { battleId: string }) {
  const { battle } = useLiveBattle(battleId)
  const queryClient = useQueryClient()
  const ended = battle.data ? !isActive(battle.data.status) : false

  useEffect(() => {
    if (!ended) return
    void queryClient.invalidateQueries({ queryKey: battlesQuery.queryKey, exact: true })
    void queryClient.invalidateQueries({ queryKey: statsQuery.queryKey })
  }, [ended, queryClient])

  const signedOut = isApiError(battle.error, 401)
  useEffect(() => {
    if (signedOut) void queryClient.invalidateQueries(meQuery)
  }, [signedOut, queryClient])

  if (battle.isPending) return <PageLoader label="Loading battle" />
  if (!battle.data) {
    const missing = isApiError(battle.error, 404)
    return (
      <div className="grid gap-4">
        <PageHeader title={missing ? 'Battle not found' : 'Battle unavailable'} />
        <p role="alert" className="text-alarm">
          {missing
            ? 'This battle does not exist, or you are not playing in it.'
            : battle.error.message}
        </p>
        <Button variant="secondary" asChild className="w-fit">
          <Link to="/">Back to play</Link>
        </Button>
      </div>
    )
  }

  const data = battle.data
  const scored = data.status === 'ONGOING' || data.status === 'FINISHED'
  return (
    <div className="grid gap-6">
      <PageHeader
        title={data.opponent ? `vs ${data.opponent.username}` : 'New battle'}
        description={describe(data)}
        badge={<StatusBadge status={data.status} outcome={data.outcome} />}
        action={isActive(data.status) && <ForfeitButton battle={data} />}
      />
      <div role="status" className="live">
        {battle.isError && (
          <p className="feedback">
            {battle.error instanceof TypeError
              ? 'Connection lost. Retrying…'
              : 'Could not refresh the battle. Retrying…'}
          </p>
        )}
      </div>
      {scored && <Scoreboard battle={data} />}
      <BattleStage battle={data} receivedAt={battle.dataUpdatedAt} />
    </div>
  )
}

function battlePhase(battle: Battle) {
  if (battle.status === 'PENDING') return { key: 'waiting', label: 'Waiting for an opponent.' }
  if (battle.status === 'SELECTING')
    return battle.myLineupLocked
      ? { key: 'locked', label: 'Lineup locked. Waiting for the first round.' }
      : { key: 'picking', label: 'Pick your song lineup.' }
  if (battle.status !== 'ONGOING')
    return { key: battle.status, label: 'Battle ended. Results are ready.' }
  const round = battle.round
  return {
    key: `${round?.id}:${round?.myFinished}`,
    label: round?.myFinished
      ? `Round ${round.number} complete for you. Waiting for your opponent.`
      : `Round ${round?.number ?? battle.currentRound}. Guess the song.`,
  }
}

function BattleStage({ battle, receivedAt }: { battle: Battle; receivedAt: number }) {
  const phase = battlePhase(battle)
  const previous = useRef(phase.key)
  const region = useRef<HTMLElement>(null)
  useLayoutEffect(() => {
    if (previous.current !== phase.key) {
      const active = document.activeElement
      if (active === document.body || region.current?.contains(active))
        region.current?.focus({ preventScroll: true })
    }
    previous.current = phase.key
  }, [phase.key])
  return (
    <section ref={region} tabIndex={-1} aria-label={phase.label}>
      <p role="status" className="sr-only">
        {phase.label}
      </p>
      <BattleBody battle={battle} receivedAt={receivedAt} />
    </section>
  )
}

function BattleBody({ battle, receivedAt }: { battle: Battle; receivedAt: number }) {
  switch (battle.status) {
    case 'PENDING':
      return <WaitingRoom battle={battle} />
    case 'SELECTING':
      return <LineupPicker battle={battle} />
    case 'ONGOING':
      return <RoundPlay battle={battle} receivedAt={receivedAt} />
    default:
      return <Results battle={battle} />
  }
}
